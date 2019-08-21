# promscriptsgw
`promscriptsgw` is a [Prometheus](https://prometheus.io/) metrics server that collects metrics and values by executing user provided scripts. It's intended to help integrate metrics from any source without requiring the user to write a custom metrics server.

Metrics are collected by executing scripts in a directory and parsing their output. Scripts are expected to output at least one line in the format
```
<metric_name>: <metric_value>
```
to stdout. `metric_name` needs to be a valid prometheus identifier, `metric_value` needs to be a float.
Currently only `gauge` metrics are supported.

## Configuration
Currently configuration can only be done through the command line. Refer to the output of `-help` for details on the available parameters.