#!/bin/bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

docker run \
    --name prom \
    -it \
    --rm \
    -v ${DIR}/prometheus.yml:/prometheus.yml \
    -p 9090:9090 \
    prom/prometheus --config.file /prometheus.yml