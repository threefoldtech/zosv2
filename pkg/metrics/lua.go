package metrics

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
	"github.com/threefoldtech/zos/pkg/metrics/generated"
)

type luaStorage struct {
	pool *redis.Pool
	hash string
}

/*
{'h_avg': 27.755708424035,
 'h_total': 138.77854212018,
 'm_max': 100,
 'h_previous': {'max': 0, 'timestamp': 0, 'avg': 0},
 'm_total': 138.77854212018,
 'm_previous': {'max': 0, 'timestamp': 0, 'avg': 0},
 'last_update': '1612259527',
 'h_nr': 5,
 'h_timestamp': 1612260000,
 'h_max': 100,
 'm_timestamp': 1612259700,
 'm_nr': 5,
 'm_avg': 27.755708424035}
*/

// lua metric loads part of the structured
// stored by the lua script in redis to return the averages over
// the previous aggregated 5min and 1 hour
type luaMetric struct {
	M struct {
		Timestamp int64   `json:"timestamp"`
		Average   float64 `json:"avg"`
	} `json:"m_previous"`
	H struct {
		Timestamp int64   `json:"timestamp"`
		Average   float64 `json:"avg"`
	} `json:"h_previous"`
}

// NewLuaStorage return a new metric storage that aggregate on redis side
// using an embedded lua script
func NewLuaStorage(address string) (Storage, error) {
	pool, err := newRedisPool(address)
	if err != nil {
		return nil, err
	}

	con := pool.Get()
	defer con.Close()

	hash, err := redis.String(con.Do("SCRIPT", "LOAD", generated.EmbeddedFiles["stat.lua"]()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to upload script")
	}

	return &luaStorage{
		pool: pool,
		hash: hash,
	}, nil
}

func (l *luaStorage) Update(name, id string, mode aggregated.AggregationMode, value float64) (float64, error) {
	con := l.pool.Get()
	defer con.Close()
	m := "A"
	if mode == aggregated.DifferentialMode {
		m = "D"
	}

	log.Debug().Str("key", name).Str("id", id).Float64("value", value).Msg("reporting")
	str, err := redis.String(con.Do("EVALSHA", l.hash, 2, name, id, m, value))
	if err != nil {
		return 0, errors.Wrap(err, "failed to update stored metrics")
	}

	return strconv.ParseFloat(str, 64)
}

func (l *luaStorage) get(con redis.Conn, key string) (Metric, error) {
	parts := strings.Split(key, ":")
	if len(parts) != 3 {
		return Metric{}, fmt.Errorf("invalid metric key")
	}

	var m luaMetric
	data, err := redis.Bytes(con.Do("GET", key))
	if err != nil {
		return Metric{}, errors.Wrap(err, "failed to retrieve metric from redis")
	}

	if err := json.Unmarshal(data, &m); err != nil {
		return Metric{}, errors.Wrap(err, "failed to load metric from redis")
	}

	return Metric{
		ID: parts[2],
		Values: []Average{
			{
				Width:     300, // always 5 min
				Timestamp: m.M.Timestamp,
				Value:     m.M.Average,
			},
			{
				Width:     3600, // always 1 hour
				Timestamp: m.H.Timestamp,
				Value:     m.H.Average,
			},
		},
	}, nil
}

func (l *luaStorage) Metrics(name string) ([]Metric, error) {
	match := fmt.Sprintf("metric:%s:*", name)
	con := l.pool.Get()
	defer con.Close()

	var metrics []Metric
	var cursor uint64 = 0
	for {
		values, err := redis.Values(con.Do("SCAN", cursor, "match", match))
		if err != nil {
			return nil, err
		}
		var keys []string
		if _, err := redis.Scan(values, &cursor, &keys); err != nil {
			return nil, errors.Wrap(err, "failed to scan matching keys")
		}

		for _, key := range keys {
			metric, err := l.get(con, key)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get metric: '%s'", key)
			}
			metrics = append(metrics, metric)
		}

		if cursor == 0 {
			break
		}
	}

	return metrics, nil
}
