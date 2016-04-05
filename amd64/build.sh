#!/bin/bash

set -ex

apt-get update -qq
apt-get install -qq -y make git golang

cd /usr/local/src/github.com/apeltzer/deager/
go get -d
go build -o amd64/deager

echo "La Fin!"
