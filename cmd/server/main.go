package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
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
		"`interval` for script execution / metrics collection")

	flag.StringVar(
		&flagScriptDir,
		"script-dir",
		"",
		"the program will look for executables in `directory` (required)")

	flag.DurationVar(
		&flagScriptTimeout,
		"script-timeout",
		1*time.Second,
		"any executable running for longer than the `timeout` value will be killed")

	flag.StringVar(
		&flagMetricsNamespace,
		"metrics-namespace",
		"",
		"sets the `namespace` portion of the name for all the metrics created by this program")

	flag.StringVar(
		&flagMetricsSubsystem,
		"metrics-subsystem",
		"",
		"sets the `subsystem` portion of the  name for all the metrics created by this program")

	flag.StringVar(
		&flagListenAddr,
		"listen-addr",
		":9000",
		"`address` the server will listen on")

	flag.Parse()

	if flagScriptDir == "" {
		log.Fatal("-script-dir is required")
	}

	if !checkMetricName(flagMetricsNamespace) {
		log.Fatalf("namespace \"%s\" is not a valid component for a prometheus metric name", flagMetricsNamespace)
	}
	if !checkMetricName(flagMetricsSubsystem) {
		log.Fatalf("subsystem \"%s\" is not a valid component for a prometheus metric name", flagMetricsSubsystem)
	}

	promRegistry = prometheus.NewRegistry()
	gauges = make(map[string]*prometheus.GaugeVec)
	runtimeGaugeName = "script_runtime_seconds"
}

func checkMetricName(n string) bool {
	if n == "" {
		return true
	}

	match, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", n)
	return match
}

func parseScriptOutputLine(line string) (name string, value float64, err error) {
	lineStrip := strings.Trim(line, "\n")

	split := strings.Split(lineStrip, ":")
	if len(split) != 2 {
		return "", 0, fmt.Errorf("\"%s\" does not match expected format", lineStrip)
	}

	if !checkMetricName(split[0]) {
		return "", 0, fmt.Errorf("\"%s\" is not a valid prometheus metric name", split[0])
	}

	name = split[0]
	value, err = strconv.ParseFloat(strings.TrimSpace(split[1]), 64)
	return
}

type collectorValue struct {
	name  string
	value float64
}

func createAndRegisterGauge(name, help string) *prometheus.GaugeVec {
	g, ok := gauges[name]
	if !ok {
		g = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: flagMetricsNamespace,
				Subsystem: flagMetricsSubsystem,
				Name:      name,
				Help:      help,
			},
			[]string{"script_name"},
		)

		err := promRegistry.Register(g)
		if err != nil {
			log.Errorf("unable to register collector: %s", err.Error())
		} else {
			log.Debugf("registered new gauge vector collector for \"%s\"", name)
			gauges[name] = g
		}
	}

	return g
}

func updateMetric(r scripts.ExecResult) error {
	var g *prometheus.GaugeVec

	stdout, err := ioutil.ReadAll(&r.Stdout)
	if err != nil {
		return err
	}

	// add a gauge for script runtime
	g = createAndRegisterGauge(runtimeGaugeName, "script runtime in seconds")
	if g == nil {
		log.Error("uh oh")
	}
	g.WithLabelValues(r.Command).Set(time.Now().Sub(r.StartTime).Seconds())

	for _, l := range strings.Split(string(stdout), "\n") {
		if l == "" {
			continue
		}

		n, v, err := parseScriptOutputLine(l)
		if err != nil {
			return err
		}

		g = createAndRegisterGauge(n, "")
		g.WithLabelValues(r.Command).Set(v)
	}

	return nil
}

// runs scripts, parse output, create the appropriate collectors
// and update the values
func executeAndCollect() {
	ss, _ := scripts.List(flagScriptDir)
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
				slog.Errorf("  - %s", r.Err.Error())
				return
			}
			slog.Debugf("  - finished successfully. ran for %s", time.Now().Sub(r.StartTime))

			// update the metric with the script result
			err := updateMetric(r)
			if err != nil {
				slog.Errorf("  - cannot update metric: %s", err.Error())
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
				log.Debug("running scripts and updating metrics")
				executeAndCollect()
			}
		}

	}()
}

func main() {
	log.SetLevel(log.DebugLevel)

	log.Infof("looking for executable scripts in \"%s\"", flagScriptDir)

	collectionLoop()
	http.Handle("/metrics", promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{
		ErrorLog:      log.StandardLogger(),
		ErrorHandling: 1,
	}))

	log.Infof("listening for requests on %s", flagListenAddr)
	log.Fatal(http.ListenAndServe(flagListenAddr, nil))
}
