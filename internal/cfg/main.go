package cfg

import (
	"flag"
	"fmt"
	"regexp"
	"time"
)

var (
	// RuntimeGaugeName is the full name of the script runtime metric
	RuntimeGaugeName string

	// CollectionInterval specifies how often scripts are executed
	// and values are collected
	CollectionInterval time.Duration

	// ScriptDir is the directory that is searched for scripts to execute
	ScriptDir string

	// ScriptTimeout is the maximum allowed runtime of any executed script
	ScriptTimeout time.Duration

	// MetricsNamespace allows setting a global namespace that is prepended
	// to all exposed metrics
	MetricsNamespace string

	// MetricsSubsystem allows setting a global subsystem that is inserted
	// between the namespace (if specified) and the metric name
	MetricsSubsystem string

	// ListenAddr is the address the server will listen on
	ListenAddr string

	// MetricsURL is the URL the metrics will be exposed
	MetricsURL string

	// Debug is a flag enabling more verbose log output
	Debug bool
)

func init() {
	// default values for config items that have no way of setting them
	// otherwise currently
	RuntimeGaugeName = "script_runtime_seconds"
	MetricsURL = "/metrics"
}

// CheckMetricName decides whether a supplied string is a valid
// name for a prometheus metric
func CheckMetricName(n string) bool {
	if n == "" {
		return true
	}

	match, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", n)
	return match
}

// FromCommandline populates select configuration values from the commandline
func FromCommandline() error {
	var (
		flagCollectionInterval time.Duration
		flagScriptDir          string
		flagScriptTimeout      time.Duration
		flagMetricsNamespace   string
		flagMetricsSubsystem   string
		flagListenAddr         string
		flagDebug              bool
	)

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

	flag.BoolVar(
		&flagDebug,
		"debug",
		false,
		"increase log verbosity")

	flag.Parse()

	if flagScriptDir == "" {
		return fmt.Errorf("-script-dir is required")
	}

	if !CheckMetricName(flagMetricsNamespace) {
		return fmt.Errorf("namespace \"%s\" is not a valid component for a prometheus metric name", flagMetricsNamespace)
	}
	if !CheckMetricName(flagMetricsSubsystem) {
		return fmt.Errorf("subsystem \"%s\" is not a valid component for a prometheus metric name", flagMetricsSubsystem)
	}

	CollectionInterval = flagCollectionInterval
	ScriptDir = flagScriptDir
	ScriptTimeout = flagScriptTimeout
	MetricsNamespace = flagMetricsNamespace
	MetricsSubsystem = flagMetricsSubsystem
	ListenAddr = flagListenAddr
	Debug = flagDebug

	return nil

}

func init() {}
