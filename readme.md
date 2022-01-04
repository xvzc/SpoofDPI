# SpoofDPI

A simple and fast software designed to bypass **Deep Packet Inspection**  
![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

See in other Languages: [English](https://github.com/xvzc/SpoofDPI), [한국어](https://github.com/xvzc/SpoofDPI/blob/main/readme_ko.md)

# Dependencies
- Go

# Installation
- With **go install**  
`$ go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi`  
  > Remember that $GOPATH variable should be set in your $PATH

- Or you can build your own  
`$ git clone https://github.com/xvzc/SpoofDPI.git`  
`$ cd SpoofDPI`  
`$ go build ./cmd/...`  

# Run
### OSX
Run `$ spoof-dpi`  

### Linux
Run `$ spoof-dpi` and open your favorite browser with proxy option  
`google-chrome --proxy-server="http://127.0.0.1:8080"`

### Windows
Use [GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) instead

# Usage
```
Usage: spoof-dpi [options...]
-dns=<addr>  | default: 8.8.8.8
-port=<port> | default: 8080
```

# How it works
### HTTP
Since most of websites in the world now support HTTPS, SpoofDPI doesn't bypass Deep Packet Inspections for HTTP requets, However It still serves proxy connection for all HTTP requests.

### HTTPS
 Although the HTTPS requests are encryted with TLS, the domains are still shown as plaintext in the encryted requests. 
 In other words, when someone else looks on a packet, they can easily identify where the packet is headed to.
 I had tried some ways to bypass the inspections, and found out that it seems like only the first chunk is inspected when we send the encryted request in chunks. 
 What SpoofDPI does to bypass this is to send the first 1 byte of a request to the server, 
 and then send the rest.
 > SpoofDPI doesn't decrypt your HTTPS requests, and that's why we don't need the SSL certificates.

# Inspiration
[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel)  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI)
