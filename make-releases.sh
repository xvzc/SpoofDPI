#!/bin/bash

VERSION="v0.10.4"

for osarch in 'darwin/amd64' 'linux/amd64' 'linux/arm' 'linux/arm64' 'linux/mips' 'linux/mipsle'; do
    GOOS=${osarch%/*} GOARCH=${osarch#*/} go build -ldflags="-w -s -X main.VERSION=${VERSION}" github.com/xvzc/SpoofDPI/cmd/spoof-dpi &&
        tar -zcvf spoof-dpi-${osarch%/*}-${osarch#*/}.tar.gz ./spoof-dpi &&
        rm -rf ./spoof-dpi
done
