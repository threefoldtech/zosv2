package monitord

import (
	"context"
	"fmt"
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
	zbucCl zbus.Client
	pusher *promPusher.Pusher
}

// NewPrometheusSystemMonitor creates a new prometheus connected system monitor
func NewPrometheusSystemMonitor(zbucCl zbus.Client, promURL string) PrometheusSystemMonitor {
	pusher := promPusher.New(promURL, "node1")

	return PrometheusSystemMonitor{
		zbucCl: zbucCl,
		pusher: pusher,
	}
}

// Run runs the prometheus system monitor jobs
func (m *PrometheusSystemMonitor) Run(ctx context.Context) error {
	log.Info().Msg("running monitord")
	// storaged := stubs.NewStorageModuleStub(m.zbucCl)

	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			log.Info().Msg("Going to scrape")
			err := m.fetchNodeStats(ctx)
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

func (m *PrometheusSystemMonitor) fetchNodeStats(ctx context.Context) error {
	resp, err := http.Get("http://localhost:9100/metrics")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	decoder := expfmt.NewDecoder(resp.Body, expfmt.FmtOpenMetrics)

	var items io_prometheus_client.MetricFamily
	err = decoder.Decode(&items)
	if err != nil {
		return err
	}

	fmt.Println(items)

	// if resp.StatusCode == http.StatusOK {
	// 	bodyBytes, err := ioutil.ReadAll(resp.Body)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	items := strings.Split(string(bodyBytes), "\n")
	// 	for _, item := range items {
	// 		v, err := expfmt.ExtractSamples(&expfmt.DecodeOptions{}, item)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		log.Info().Msg(v.String())
	// 	}
	// }

	return nil
}
