package main

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/threefoldtech/zos/pkg/metrics"
	"github.com/threefoldtech/zos/pkg/metrics/collectors"
)

func writeHeader(writer *bufio.Writer, metric *collectors.Metric) {
	writer.WriteString("#HELP ")
	writer.WriteString(metric.Name)
	writer.WriteRune(' ')
	writer.WriteString(metric.Descritpion)
	writer.WriteRune('\n')
}

func writeError(writer *bufio.Writer, metric *collectors.Metric, err error) {
	writer.WriteString("#ERROR failed to get values for metric ")
	writer.WriteString(metric.Name)
	writer.WriteRune(' ')
	writer.WriteString(err.Error())
	writer.WriteRune('\n')
}

func createServeMux(storage metrics.Storage, metrics []collectors.Metric) *http.ServeMux {
	server := http.NewServeMux()

	server.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		writer := bufio.NewWriter(w)
		defer writer.Flush()

		for _, metric := range metrics {
			writeHeader(writer, &metric)

			values, err := storage.Metrics(metric.Name)
			if err != nil {
				writeError(writer, &metric, err)
				continue
			}

			for _, value := range values {
				writer.WriteString(metric.Name)
				writer.WriteRune('.')
				writer.WriteString(value.ID)
				for _, v := range value.Values {
					writer.WriteRune(' ')
					writer.WriteString(fmt.Sprintf("%f", v))
				}
				writer.WriteByte('\n')
			}
		}
	})

	return server
}
