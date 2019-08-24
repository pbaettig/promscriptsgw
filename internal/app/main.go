package app

import (
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pbaettig/promscriptsgw/internal/cfg"
	"github.com/pbaettig/promscriptsgw/internal/pkg/scripts"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	// Registry is where all of the prometheus metrics are registered to
	Registry *prometheus.Registry

	gauges map[string]*prometheus.GaugeVec
)

func init() {
	Registry = prometheus.NewRegistry()
	gauges = make(map[string]*prometheus.GaugeVec)
}

func parseScriptOutputLine(line string) (name string, value float64, err error) {
	lineStrip := strings.Trim(line, "\n")

	split := strings.Split(lineStrip, ":")
	if len(split) != 2 {
		return "", 0, fmt.Errorf("\"%s\" does not match expected format", lineStrip)
	}

	if !cfg.CheckMetricName(split[0]) {
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
				Namespace: cfg.MetricsNamespace,
				Subsystem: cfg.MetricsSubsystem,
				Name:      name,
				Help:      help,
			},
			[]string{"script_name"},
		)

		err := Registry.Register(g)
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
	g = createAndRegisterGauge(cfg.RuntimeGaugeName, "script runtime in seconds")
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
	ss, _ := scripts.List(cfg.ScriptDir)
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
			ctx, cancel := context.WithTimeout(bgCtx, cfg.ScriptTimeout)
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

func CollectionLoop(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(cfg.CollectionInterval)

	wg.Add(1)
	go func() {
		defer ticker.Stop()
		defer wg.Done()

		// first run
		executeAndCollect()

		for {
			select {
			case <-ticker.C:
				// periodically execute
				log.Debug("running scripts and updating metrics")
				executeAndCollect()

			case <-ctx.Done():
				log.Debug("exiting collection loop")
				return
			}
		}
	}()
}
