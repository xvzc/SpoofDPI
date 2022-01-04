# SpoofDPI

A simple and fast software designed to bypass Deep Packet Inspection  
![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

# Installation

## Build / Installation

Install with **go install**  
`$ go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi`  
> Remember that $GOPATH variable should be set in your $PATH

Or you can build your own  
`$ go build ./cmd/...`  

# Run
## OSX
`sh ./on.sh`  
`go run ./cmd/spoof-dpi/main.go`  
`sh ./off.sh`  

## Linux
open your favorite browser with proxy option  
`google-chrome --proxy-server="http://127.0.0.1:8080"`

