#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

if [[ -z $1 ]]; then
    rev="master"
else
    rev=$1
fi

if [[ $rev == "master" ]]; then
    version="$(git rev-parse --short HEAD)"
else
    version=$rev
fi

executable_name="promscriptsgw"

tmpdir=$(mktemp -d)
project_path="$tmpdir/src/github.com/pbaettig/promscriptsgw"

mkdir -p $project_path
cd $project_path

git clone git@github.com:pbaettig/promscriptsgw.git .
git checkout $rev


GOPATH=$tmpdir go get \
    github.com/sirupsen/logrus \
    github.com/prometheus/client_golang/prometheus/promhttp \
    github.com/prometheus/client_golang/prometheus \


cd cmd/promscriptsgw

sed -r -i "s/version = \".+\"/version = \"$version\"/g" version.go

cat version.go
GOPATH=$tmpdir go build -o $executable_name *.go 

cp $executable_name $DIR/

rm -rf $tmpdir