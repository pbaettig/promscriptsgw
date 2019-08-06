#!/bin/bash

docker run \
    --name prom \
    -it \
    --rm \
    -v $(pwd)/prometheus.yml:/prometheus.yml \
    -p 9090:9090 \
    prom/prometheus --config.file /prometheus.yml