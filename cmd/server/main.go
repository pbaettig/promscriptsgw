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
	promRegistry           *prometheus.Registry
	gauges                 map[string]*prometheus.GaugeVec
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

	if flagMetricsSubsystem == "" {
		hn, err := os.Hostname()
		if err != nil {
			log.Fatal("unable to determine system hostname, set -metrics-subsystem to disable hostname lookup")
		}
		flagMetricsSubsystem = hn
	}

	promRegistry = prometheus.NewRegistry()
	gauges = make(map[string]*prometheus.GaugeVec)

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

func updateMetric(r scripts.ExecResult) error {
	stdout, err := ioutil.ReadAll(&r.Stdout)
	if err != nil {
		return err
	}

	for _, l := range strings.Split(string(stdout), "\n") {
		if l == "" {
			continue
		}

		n, v, err := parseScriptOutputLine(l)
		if err != nil {
			log.Warnf("cannot convert script output \"%s\"", l)
			return err
		}

		g, ok := gauges[n]
		if !ok {
			log.Debugf("registering new gauge vector collector for \"%s\"", n)
			g = prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					// Namespace: "our_company",
					// Subsystem: "blob_storage",
					Name: n,
					Help: "",
				},
				[]string{
					"script_name",
				},
			)
			gauges[n] = g

			err := promRegistry.Register(g)
			if err != nil {
				log.Errorf("unable to register gauge \"%s\"", n)
			}
		}
		g.WithLabelValues(r.Command).Set(v)
	}

	return nil
}

func executeAndCollect() {
	ss, _ := scripts.List("/tmp/scripts")
	bgCtx := context.Background()
	wg := new(sync.WaitGroup)

	// var buf scripts.MutexedBuffer

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

			err := updateMetric(r)
			if err != nil {

			}

			// buf.Mutex.Lock()
			// defer buf.Mutex.Unlock()

			// r.Stdout.WriteTo(&buf.Buf)
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
		ErrorHandling: 1,
	}))

	log.Fatal(http.ListenAndServe(":9000", nil))
}
