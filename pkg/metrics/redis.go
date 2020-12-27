package metrics

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
)

const (
	redisPullTimeout = 10
	redisResponseTTL = 5 * 60 // 5 minutes
)

type redisStorage struct {
	pool      *redis.Pool
	durations []time.Duration
}

func newRedisPool(address string) (*redis.Pool, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	var host string
	switch u.Scheme {
	case "tcp":
		host = u.Host
	case "unix":
		host = u.Path
	default:
		return nil, fmt.Errorf("unknown scheme '%s' expecting tcp or unix", u.Scheme)
	}
	var opts []redis.DialOption

	if u.User != nil {
		opts = append(
			opts,
			redis.DialPassword(u.User.Username()),
		)
	}

	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial(u.Scheme, host, opts...)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) > 10*time.Second {
				//only check connection if more than 10 second of inactivity
				_, err := c.Do("PING")
				return err
			}

			return nil
		},
		MaxActive:   10,
		IdleTimeout: 1 * time.Minute,
		Wait:        true,
	}, nil
}

// NewRedisStorage creates a new redis backed metrics aggregator
func NewRedisStorage(address string, durations ...time.Duration) (Storage, error) {
	pool, err := newRedisPool(address)
	if err != nil {
		return nil, err
	}

	return &redisStorage{pool, durations}, nil
}

func (r *redisStorage) key(name, id string) string {
	return fmt.Sprintf("metric:%s:%s", name, id)
}

func (r *redisStorage) Update(name, id string, mode aggregated.AggregationMode, value float64) error {
	if strings.Contains(name, ":") || strings.Contains(id, ":") {
		return fmt.Errorf("unicode ':' is reserved")
	}

	key := r.key(name, id)
	con := r.pool.Get()
	defer con.Close()

	var ag aggregated.Aggregated
	data, err := redis.Bytes(con.Do("GET", key))
	if err == redis.ErrNil {
		ag = aggregated.NewAggregatedMetric(mode, r.durations...)
	} else if err != nil {
		return errors.Wrap(err, "failed to retrieve metric from redis")
	} else {
		if err := json.Unmarshal(data, &ag); err != nil {
			return errors.Wrap(err, "failed to load metric from redis")
		}
	}

	if ag.Mode != mode {
		return fmt.Errorf("unexpected aggregation mode '%d' for metric '%s' expected '%d'", mode, name, ag.Mode)
	}

	ag.Sample(value)
	data, err = json.Marshal(&ag)
	if err != nil {
		return errors.Wrap(err, "failed to serialize metric")
	}

	if _, err := con.Do("SET", key, data); err != nil {
		return errors.Wrap(err, "failed to serialize metric")
	}

	return nil
}

func (r *redisStorage) get(con redis.Conn, key string) (Metric, error) {
	parts := strings.Split(key, ":")
	if len(parts) != 3 {
		return Metric{}, fmt.Errorf("invalid metric key")
	}

	var ag aggregated.Aggregated
	data, err := redis.Bytes(con.Do("GET", key))
	if err != nil {
		return Metric{}, errors.Wrap(err, "failed to retrieve metric from redis")
	}

	if err := json.Unmarshal(data, &ag); err != nil {
		return Metric{}, errors.Wrap(err, "failed to load metric from redis")
	}

	return Metric{
		ID:     parts[2],
		Values: ag.Averages(),
	}, nil
}

func (r *redisStorage) Metrics(name string) ([]Metric, error) {
	match := fmt.Sprintf("metric:%s:*", name)
	con := r.pool.Get()
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
			metric, err := r.get(con, key)
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
