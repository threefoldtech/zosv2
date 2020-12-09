package main

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
)

func main() {

	storage, err := metrics.NewRedisStorage("tcp://localhost:6379", 1*time.Minute, 5*time.Minute, time.Hour, 24*time.Hour)
	if err != nil {
		log.Fatal().Err(err)
	}

	if err := storage.Update("disk.size", "sda", aggregated.AverageMode, 38); err != nil {
		log.Error().Err(err).Msg("failed to set value")
	}

	metrics, err := storage.Metrics("disk.size")
	if err != nil {
		panic(err)
	}

	for _, m := range metrics {
		fmt.Println("- ", m.ID)
		fmt.Println("  - ", m.Values)
	}
}
