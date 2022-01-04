# SpoofDPI

A simple and fast software designed to bypass **Deep Packet Inspection**  
![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

# Installation
- With **go install**  
`$ go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi`  
  > Remember that $GOPATH variable should be set in your $PATH

- Or you can build your own  
`$ git clone https://github.com/xvzc/SpoofDPI.git`  
`$ cd SpoofDPI`  
`$ go build ./cmd/...`  

# Run
## OSX
`$ spoof-dpi`  

## Linux
Open your favorite browser with proxy option  
`google-chrome --proxy-server="http://127.0.0.1:8080"`

## Windows
Use [GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) instead

# Usage
```
Usage: spoof-dpi [options...]
-dns=<addr>  | default: 8.8.8.8
-port=<port> | default: 8080
```

# Inspiration
[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel)  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI)
