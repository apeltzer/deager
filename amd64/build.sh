#!/bin/bash

set -ex

dpkg --configure -a
apt-get update -qq
apt-get install -qq -y make git golang

cd /usr/local/src/github.com/apeltzer/deager/
go get -d
go build -o amd64/deager

echo "La Fin!"
