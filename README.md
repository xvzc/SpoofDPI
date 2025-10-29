# SpoofDPI

Read in other Languages: [🇬🇧English](https://github.com/xvzc/SpoofDPI), [🇰🇷한국어](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ko.md), [🇨🇳简体中文](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_zh-cn.md), [🇷🇺Русский](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ru.md), [🇯🇵日本語](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ja.md)

A simple and fast software designed to bypass **Deep Packet Inspection**.


```txt
 ❯ spoofdpi
███████ ██████   ██████   ██████  ███████ ██████  ██████  ██
██      ██   ██ ██    ██ ██    ██ ██      ██   ██ ██   ██ ██
███████ ██████  ██    ██ ██    ██ █████   ██   ██ ██████  ██
     ██ ██      ██    ██ ██    ██ ██      ██   ██ ██      ██
███████ ██       ██████   ██████  ██      ██████  ██      ██

• ADDR    : 127.0.0.1
• PORT    : 8080
• DNS     : 8.8.8.8
• DEBUG   : false
```

# Installation
See the installation guide for SpoofDPI [here](https://github.com/xvzc/SpoofDPI/blob/main/_docs/INSTALL.md).

<a href="https://repology.org/project/spoofdpi/versions">
    <img src="https://repology.org/badge/vertical-allrepos/spoofdpi.svg?columns=1" alt="Packaging status">
</a>  

# Usage
```
Usage: spoofdpi [options...]
  -addr string
        listen address (default "127.0.0.1")
  -debug
        enable debug output
  -dns-addr string
        dns address (default "8.8.8.8")
  -dns-ipv4-only
        resolve only version 4 addresses
  -dns-port value
        port number for dns (default 53)
  -enable-doh
        enable 'dns-over-https'
  -pattern value
        bypass DPI only on packets matching this regex pattern; can be given multiple times
  -port value
        port (default 8080)
  -silent
        do not show the banner and server information at start up
  -system-proxy
        enable system-wide proxy (default true)
  -timeout value
        timeout in milliseconds; no timeout when not given
  -v    print spoofdpi's version; this may contain some other relevant information
  -window-size value
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
