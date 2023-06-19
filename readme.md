**⭐PRs or any form of contribution will be appreciated⭐**

# SpoofDPI

Read in other Languages: [🇬🇧English](https://github.com/xvzc/SpoofDPI), [🇰🇷한국어](https://github.com/xvzc/SpoofDPI/blob/main/readme_ko.md), [🇨🇳简体中文](https://github.com/xvzc/SpoofDPI/blob/main/readme_zh-cn.md), [🇷🇺Русский](https://github.com/xvzc/SpoofDPI/blob/main/readme_ru.md)

A simple and fast software designed to bypass **Deep Packet Inspection**  
  
![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

# Installation
## Binary
SpoofDPI will be installed in `~/.spoof-dpi/bin`.  
To run SpoofDPI in any directory, add the line below to your `~/.bashrc || ~/.zshrc || ...`
```
export PATH=$PATH:~/.spoof-dpi/bin
```

### curl
Install the latest binary with curl
- OSX
```
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s osx
```
- Linux
```
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux
```
### wget
Install the latest binary with wget
- OSX
```
wget -O - https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s osx 
```
- Linux
```
wget -O - https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux 
```
## Go
You can also install SpoofDPI with **go install**  
`$ go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi`  
  > Remember that $GOPATH should be set in your $PATH

## Git
You can also build your own  
`$ git clone https://github.com/xvzc/SpoofDPI.git`  
`$ cd SpoofDPI`  
`$ go build ./cmd/...`  

# Usage
```
Usage: spoof-dpi [options...]
--addr=<addr>       | default: 127.0.0.1
--dns=<addr>        | default: 8.8.8.8
--port=<port>       | default: 8080
--debug=<bool>      | default: false
--banner=<bool>     | default: true
--url=<url>         | Can be used multiple times. If set, 
                    | it will bypass DPI only for this url. 
                    | Example: --url=google.com --url=github.com
--pattern=<regex>   | If set, it will bypass DPI only for packets 
                    | that matches this regex pattern.
                    | Example: --pattern="google|github"
```
> If you are using any vpn extensions such as Hotspot Shield in Chrome browser,   
  go to Settings > Extensions, and disable them.

### OSX
Run `$ spoof-dpi` and it will automatically set your proxy

### Linux
Run `$ spoof-dpi` and open your favorite browser with proxy option  
`google-chrome --proxy-server="http://127.0.0.1:8080"`

# How it works
### HTTP
Since most of websites in the world now support HTTPS, SpoofDPI doesn't bypass Deep Packet Inspection for HTTP requets, However It still serves proxy connection for all HTTP requests.

### HTTPS
 Although TLS 1.3 encrypts every handshake process, the domain names are still shown as plaintext in the Client hello packet. 
 In other words, when someone else looks on the packet, they can easily guess where the packet is headed to. 
 The domain name can offer a significant information while DPI is being processed, and we can actually see that the connection is blocked right after sending Client hello packet.
 I had tried some ways to bypass this, and found out that it seemed like only the first chunk gets inspected when we send the Client hello packet splited in chunks. 
 What SpoofDPI does to bypass this is to send the first 1 byte of a request to the server, 
 and then send the rest.
 > SpoofDPI doesn't decrypt your HTTPS requests, and that's why we don't need the SSL certificates.

# Inspirations
[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel) by @SadeghHayeri  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) by @ValdikSS



