package monitord

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/collectors"
	"github.com/urfave/cli/v2"
)

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "monitord",
	Usage: "start metrics aggregation",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
		&cli.BoolFlag{
			Name:  "dump",
			Usage: "dumps current metrics and exit",
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
		dump         bool   = cli.Bool("dump")
	)

	cl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to connect to redis")
	}

	//storage, err := metrics.NewRedisStorage(msgBrokerCon, 1*time.Minute, 5*time.Minute, time.Hour, 24*time.Hour)
	storage, err := metrics.NewLuaStorage(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to connect to redis")
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
				for _, metric := range values {
					fmt.Printf("  - %s: ", metric.ID) //%+v\n", value.ID, value.Values)
					for _, value := range metric.Values {
						fmt.Printf("%s ", value.String())
					}
					fmt.Println()
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
		return errors.Wrap(err, "failed to serve metrics")
	}

	return nil
}
