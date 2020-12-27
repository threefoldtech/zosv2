package metrics

import (
	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
	"github.com/threefoldtech/zos/pkg/metrics/generated"
)

type luaStorage struct {
	pool *redis.Pool
	hash string
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

func (l *luaStorage) Update(name, id string, mode aggregated.AggregationMode, value float64) error {
	con := l.pool.Get()
	defer con.Close()
	m := "A"
	if mode == aggregated.DifferentialMode {
		m = "D"
	}

	_, err := con.Do("EVALSHA", l.hash, 2, name, id, m, value)

	return err
}

func (l *luaStorage) Metrics(name string) ([]Metric, error) {
	return nil, nil
}
