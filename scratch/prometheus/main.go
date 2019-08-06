package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	promRegistry *prometheus.Registry
)

func init() {
	cpuTemp := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_temperature_celsius",
		Help: "Current temperature of the CPU.",
	})
	promRegistry = prometheus.NewRegistry()
	promRegistry.MustRegister(cpuTemp)
}

func main() {

	http.Handle("/metrics", promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{
		ErrorHandling: 1,
	}))

	log.Fatal(http.ListenAndServe(":9090", nil))
}
