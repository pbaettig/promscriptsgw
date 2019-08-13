package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/pbaettig/script-server/internal/pkg/scripts"
	log "github.com/sirupsen/logrus"
)

var (
	flagCollectionInterval time.Duration
	flagScriptDir          string
	flagScriptTimeout      time.Duration
	flagMetricsNamespace   string
	flagMetricsSubsystem   string
	flagListenAddr         string
	promRegistry           *prometheus.Registry
	gauges                 map[string]*prometheus.GaugeVec
	runtimeGaugeName       string
)

func init() {
	flag.DurationVar(
		&flagCollectionInterval,
		"collection-interval",
		10*time.Second,
		"interval for script execution / metrics collection")

	flag.StringVar(
		&flagScriptDir,
		"script-dir",
		"/tmp/scripts",
		"directory to search for scripts")

	flag.DurationVar(
		&flagScriptTimeout,
		"script-timeout",
		1*time.Second,
		"timeout for script")

	flag.StringVar(
		&flagMetricsNamespace,
		"metrics-namespace",
		"ACME Ltd.",
		"namespace for prometheus metrics")

	flag.StringVar(
		&flagMetricsSubsystem,
		"metrics-subsystem",
		"",
		"prometheus metrics subsystem")

	flag.StringVar(
		&flagListenAddr,
		"listen-addr",
		":9000",
		"ip:port for the server to listen on")

	if flagMetricsSubsystem == "" {
		hn, err := os.Hostname()
		if err != nil {
			log.Fatal("unable to determine system hostname, set -metrics-subsystem to disable hostname lookup")
		}
		flagMetricsSubsystem = hn
	}

	promRegistry = prometheus.NewRegistry()
	gauges = make(map[string]*prometheus.GaugeVec)
	runtimeGaugeName = "script_runtime_seconds"
}

func parseScriptOutputLine(line string) (name string, value float64, err error) {
	lineStrip := strings.Trim(line, "\n")

	split := strings.Split(lineStrip, ":")
	if len(split) != 2 {
		return "", 0, fmt.Errorf("\"%s\" does not match expected format", lineStrip)
	}

	name = split[0]
	value, err = strconv.ParseFloat(strings.TrimSpace(split[1]), 64)
	return
}

type collectorValue struct {
	name  string
	value float64
}

func createAndRegisterGauge(opts prometheus.GaugeOpts) *prometheus.GaugeVec {
	g, ok := gauges[opts.Name]
	if !ok {
		g = prometheus.NewGaugeVec(
			opts,
			[]string{"script_name"},
		)

		gauges[opts.Name] = g
		err := promRegistry.Register(g)
		if err != nil {
			log.Errorf("unable to register collector: %s", err.Error())
		} else {
			log.Debugf("registered new gauge vector collector for \"%s\"", opts.Name)
		}
	}

	return g
}

func updateMetric(r scripts.ExecResult) error {
	stdout, err := ioutil.ReadAll(&r.Stdout)
	if err != nil {
		return err
	}

	// add a gauge for script runtime
	g := createAndRegisterGauge(
		prometheus.GaugeOpts{
			Name: runtimeGaugeName,
			Help: "script runtime in seconds",
		},
	)
	g.WithLabelValues(r.Command).Set(time.Now().Sub(r.StartTime).Seconds())

	for _, l := range strings.Split(string(stdout), "\n") {
		if l == "" {
			continue
		}

		n, v, err := parseScriptOutputLine(l)
		if err != nil {
			log.Warnf("cannot convert script output \"%s\"", l)
			return err
		}

		g = createAndRegisterGauge(
			prometheus.GaugeOpts{
				Name: n,
				Help: "",
			},
		)

		g.WithLabelValues(r.Command).Set(v)
	}

	return nil
}

// runs scripts, parse output, create the appropriate collectors
// and update the values
func executeAndCollect() {
	ss, _ := scripts.List("/tmp/scripts")
	bgCtx := context.Background()
	wg := new(sync.WaitGroup)

	// Go through all the scripts...
	for _, sp := range ss {
		wg.Add(1)

		// Start each script asynchronously
		go func(scriptPath string) {
			slog := log.WithFields(log.Fields{
				"script": scriptPath,
			})
			ctx, cancel := context.WithTimeout(bgCtx, flagScriptTimeout)
			defer cancel()
			defer wg.Done()

			rc := scripts.RunAsync(ctx, scriptPath, []string{})
			r := <-rc

			if r.Err != nil {
				slog.Error(r.Err.Error())
				return
			}
			slog.Debugf("finished successfully. ran for %s", time.Now().Sub(r.StartTime))

			// update the metric with the script result
			err := updateMetric(r)
			if err != nil {
				log.Errorf("failed to update metric")
			}

		}(sp)
	}
	wg.Wait()
}

func collectionLoop() {
	ticker := time.NewTicker(flagCollectionInterval)

	go func() {
		defer ticker.Stop()
		executeAndCollect()

		for {
			select {
			case <-ticker.C:
				log.Debug("updating metrics")
				executeAndCollect()
			}
		}

	}()
}

func main() {
	log.SetLevel(log.DebugLevel)
	collectionLoop()
	http.Handle("/metrics", promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{
		ErrorLog:      log.StandardLogger(),
		ErrorHandling: 1,
	}))

	log.Fatal(http.ListenAndServe(":9000", nil))
}
