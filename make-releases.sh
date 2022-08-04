#!/bin/bash

GOOS=darwin GOARCH=amd64 go build -ldflags='-w -s' github.com/xvzc/SpoofDPI/cmd/spoof-dpi && tar -zcvf spoof-dpi-osx.tar.gz ./spoof-dpi && rm -rf ./spoof-dpi

GOOS=linux GOARCH=amd64 go build -ldflags='-w -s' github.com/xvzc/SpoofDPI/cmd/spoof-dpi && tar -zcvf spoof-dpi-linux.tar.gz ./spoof-dpi && rm -rf ./spoof-dpi
