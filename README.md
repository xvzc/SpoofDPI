# SpoofDPI

A simple and fast software designed to bypass **Deep Packet Inspection**.


```txt
 ❯ spoofdpi

 .d8888b.                              .d888 8888888b.  8888888b. 8888888
d88P  Y88b                            d88P"  888  "Y88b 888   Y88b  888
Y88b.                                 888    888    888 888    888  888
 "Y888b.   88888b.   .d88b.   .d88b.  888888 888    888 888   d88P  888
    "Y88b. 888 "88b d88""88b d88""88b 888    888    888 8888888P"   888
      "888 888  888 888  888 888  888 888    888    888 888         888
Y88b  d88P 888 d88P Y88..88P Y88..88P 888    888  .d88P 888         888
 "Y8888P"  88888P"   "Y88P"   "Y88P"  888    8888888P"  888       8888888
           888
           888
           888

 • LISTEN_ADDR : 127.0.0.1
 • LISTEN_PORT : 8080
 • DNS_ADDR    : 8.8.8.8
 • DNS_PORT    : 53
 • DEBUG       : false

Press 'CTRL + c' to quit
```

# Dependencies
```
go >= 1.22
libpcap
```

# Installation
```sh
# To install locally
GOBIN=~/.local/bin go install github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest

# To install system wide
GOBIN=/usr/bin go install github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest
```

# Build from the source
```sh
CGO_ENABLED=1 go build -ldflags="-w -s" ./cmd/...
```

# Usage
```
Usage: spoofdpi [options...]
  -cache-shards uint
        number of shards to use for ttlcache; it is recommended to set
        this to be >= the number of CPU cores for optimal performance (max 256) (default 32)
  -debug
        enable debug output
  -dns-addr string
        dns address (default "8.8.8.8")
  -dns-ipv4-only
        resolve only IPv4 addresses
  -dns-port uint
        port number for dns (default 53)
  -doh-endpoint string
        endpoint for 'dns over https'
  -enable-doh
        enable 'dns-over-https'
  -fake-https-packets uint
        number of fake packets to send before the client hello (default 0, max 50)
        higher values may increase success, but the lowest possible value is recommended.
        try this if tcp-level fragmentation (via --window-size) does not work.
        this feature requires root privilege and the 'libpcap' dependency
  -listen-addr string
        IP address to listen on (default "127.0.0.1")
  -listen-port uint
        port number to listen on (default 8080)
  -pattern value
        bypass DPI only on packets matching this regex pattern; can be given multiple times
  -silent
        do not show the banner and server information at start up
  -system-proxy
        enable system-wide proxy (default true)
  -timeout uint
        timeout in milliseconds; no timeout when not given
  -v    print spoofdpi's version; this may contain some other relevant information
  -window-size uint
        chunk size, in number of bytes, for fragmented client hello,
        try lower values if the default value doesn't bypass the DPI;
        when not given, the client hello packet will be sent in two parts:
        fragmentation for the first data packet and the rest
```
> If you are using any vpn extensions such as Hotspot Shield in Chrome browser,
  go to Settings > Extensions, and disable them.

### OSX
Run `spoofdpi` and it will automatically set your proxy

### Linux
Run `spoofdpi` and open your favorite browser with proxy option
```bash
google-chrome --proxy-server="http://127.0.0.1:8080"
```

# How it works
### HTTP
 Since most websites in the world now support HTTPS, SpoofDPI doesn't bypass Deep Packet Inspections for HTTP requests, However, it still serves proxy connection for all HTTP requests.

### HTTPS
 Although TLS encrypts every handshake process, the domain names are still shown as plaintext in the Client hello packet.
 In other words, when someone else looks on the packet, they can easily guess where the packet is headed to.
 The domain name can offer significant information while DPI is being processed, and we can actually see that the connection is blocked right after sending Client hello packet.
 I had tried some ways to bypass this and found out that it seemed like only the first chunk gets inspected when we send the Client hello packet split into chunks.
 What SpoofDPI does to bypass this is to send the first 1 byte of a request to the server,
 and then send the rest.

# Inspirations
[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel) by @SadeghHayeri  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) by @ValdikSS
