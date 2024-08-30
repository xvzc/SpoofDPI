# Building from Source
Although pre-built binaries are available for multiple platforms, you can also build your own binaries on your need.

## Prerequisites
1. Ensure you've installed go version `1.21`
2. Clone this repository to a location of your choice.

## Build
```bash
CGO_ENABLED=0 go build -ldflags="-w -s" ./cmd/...
```
