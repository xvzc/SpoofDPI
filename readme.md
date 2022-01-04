# SpoofDPI

A simple and fast software designed to bypass Deep Packet Inspection  

```text
███████ ██████   ██████   ██████  ███████ ██████  ██████  ██  
██      ██   ██ ██    ██ ██    ██ ██      ██   ██ ██   ██ ██  
███████ ██████  ██    ██ ██    ██ █████   ██   ██ ██████  ██  
     ██ ██      ██    ██ ██    ██ ██      ██   ██ ██      ██  
███████ ██       ██████   ██████  ██      ██████  ██      ██  
```

## Installation

### Build / Installation

#### build
Build the project with `$ go build ./cmd/...`  
Or you can install with `$ go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi`  
> Remember that $GOPATH variable is set in your $PATH

### Run(OSX)
`sh ./on.sh`  
`go run ./cmd/spoof-dpi/main.go`  
`sh ./off.sh`  

### Linux
open your favorite browser with proxy option  
`google-chrome --proxy-server="http://127.0.0.1:8080"`

