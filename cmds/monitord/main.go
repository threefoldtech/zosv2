package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/aggregated"
	"github.com/threefoldtech/zos/pkg/metrics/collectors"
	"github.com/threefoldtech/zos/pkg/version"
)

func main() {
	var (
		msgBrokerCon string
		debug        bool
		ver          bool
	)

	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	cl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}

	storage, err := metrics.NewRedisStorage(msgBrokerCon, 1*time.Minute, 5*time.Minute, time.Hour)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}

	if err := storage.Update("disk.size", "sda", aggregated.AverageMode, 38); err != nil {
		log.Error().Err(err).Msg("failed to set value")
	}

	modules := []collectors.Collector{
		collectors.NewDiskCollector(cl, storage),
		collectors.NewCPUCollector(storage),
		collectors.NewMemoryCollector(storage),
	}

	for {
		for _, collector := range modules {
			if err := collector.Collect(); err != nil {
				log.Error().Err(err).Msgf("failed to collect metrics from '%T'", collector)
			}
		}

		<-time.After(30 * time.Second)
		// DEBUG CODE:
		for _, collector := range modules {
			for _, key := range collector.Metrics() {
				values, err := storage.Metrics(key)
				if err != nil {
					log.Error().Err(err).Str("id", key).Msg("failed to get metric")
				}
				fmt.Println("key:", key)
				for _, value := range values {
					fmt.Printf(" - %s: %+v\n", value.ID, value.Values)
				}
			}
		}
	}
}
