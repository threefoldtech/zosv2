package monitord

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	promPusher "github.com/prometheus/client_golang/prometheus/push"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
)

var (
	_       pkg.SystemMonitor = (*systemMonitor)(nil)
	cpuTemp                   = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_temperature_celsius",
		Help: "Current temperature of the CPU.",
	})
	hdFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "broken_devices",
			Help: "Number of Broken devices",
		},
		[]string{"device"},
	)
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(cpuTemp)
	prometheus.MustRegister(hdFailures)
}

// PrometheusSystemMonitor is a prometheus monitor struct
type PrometheusSystemMonitor struct {
	zbucCl           zbus.Client
	pusher           *promPusher.Pusher
	nodesExporterURL string
}

// NewPrometheusSystemMonitor creates a new prometheus connected system monitor
func NewPrometheusSystemMonitor(zbucCl zbus.Client, promURL string, nodesExporterURL string) PrometheusSystemMonitor {
	pusher := promPusher.New(promURL, "node1")

	return PrometheusSystemMonitor{
		zbucCl:           zbucCl,
		pusher:           pusher,
		nodesExporterURL: nodesExporterURL,
	}
}

// Run runs the prometheus system monitor jobs
func (m *PrometheusSystemMonitor) Run(ctx context.Context) error {
	log.Info().Msg("running monitord")

	ticker := time.NewTicker(1 * time.Minute)

	for {
		select {
		case <-ticker.C:
			err, _ := m.fetchNodeStats(ctx)
			if err != nil {
				fmt.Println(err)
			}
			// cpuTemp.Set(65.3)
			// hdFailures.With(prometheus.Labels{"device": "/dev/sda"}).Inc()
		case <-ctx.Done():
			ticker.Stop()
			log.Info().Msg("monitord context done, exiting")
			return nil
		}
	}
	// // The Handler function provides a default handler to expose metrics
	// // via an HTTP server. "/metrics" is the usual endpoint for that.
	// http.Handle("/metrics", promhttp.Handler())
	// return http.ListenAndServe(":31123", nil)
}

func (m *PrometheusSystemMonitor) fetchNodeStats(ctx context.Context) ([]io_prometheus_client.MetricFamily, error) {
	resp, err := http.Get(m.nodesExporterURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metrics []io_prometheus_client.MetricFamily
	decoder := expfmt.NewDecoder(resp.Body, expfmt.FmtOpenMetrics)

	for {
		var metric io_prometheus_client.MetricFamily
		err := decoder.Decode(&metric)
		if err != nil {
			if err == io.EOF {
				break
			}
			return metrics, err
		}
		metrics = append(metrics, metric)
	}
	return metrics, nil
}
