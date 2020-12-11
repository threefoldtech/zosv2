package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
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
		dump         bool
	)

	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.BoolVar(&dump, "dump", false, "dump collected metrics and exit")
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

	storage, err := metrics.NewRedisStorage(msgBrokerCon, 1*time.Minute, 5*time.Minute, time.Hour, 24*time.Hour)
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
		collectors.NewTempsCollector(storage),
	}

	if dump {

		for _, collector := range modules {
			for _, key := range collector.Metrics() {
				values, err := storage.Metrics(key.Name)
				if err != nil {
					log.Error().Err(err).Str("id", key.Name).Msg("failed to get metric")
				}
				fmt.Printf("- %s (%s)\n", key.Name, key.Descritpion)
				for _, value := range values {
					fmt.Printf("  - %s: %+v\n", value.ID, value.Values)
				}
			}
		}

		os.Exit(0)
	}

	var metrics []collectors.Metric
	for _, collector := range modules {
		metrics = append(metrics, collector.Metrics()...)
	}

	// collection
	go func() {
		for {
			for _, collector := range modules {
				if err := collector.Collect(); err != nil {
					log.Error().Err(err).Msgf("failed to collect metrics from '%T'", collector)
				}
			}

			<-time.After(30 * time.Second)
		}
	}()

	mux := createServeMux(storage, metrics)

	server := http.Server{
		Addr:    ":9100",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal().Err(err).Msg("failed to serve metrics")
	}
}
