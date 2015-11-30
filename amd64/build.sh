#!/bin/bash

set -ex

apt-get update 
apt-get install -y make git golang

cd /usr/local/src/github.con/apeltzer/deager/
go get -d
go build -o amd64/deager

echo "La Fin!"
