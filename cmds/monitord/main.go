package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/monitord"
	"github.com/threefoldtech/zos/pkg/utils"
)

const (
	module = "monitor"
)

func main() {
	var (
		msgBrokerCon          string
		prometheusEndpoint    string
		nodesExporterEndpoint string
	)

	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.StringVar(&prometheusEndpoint, "prometheusURL", "http:://192.168.170.0:9090", "connection string to the remote prometheus server")
	flag.StringVar(&nodesExporterEndpoint, "nodesExporterEndpoint", "http:://localhost:9100", "connection string to the local nodes exporter")

	zbusCl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to message broker server")
	}

	monitor := monitord.NewPrometheusSystemMonitor(zbusCl, prometheusEndpoint, nodesExporterEndpoint)

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting monitor module")

	ctx := context.Background()
	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := monitor.Run(ctx); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
	log.Info().Msg("monitord stopped")
}
