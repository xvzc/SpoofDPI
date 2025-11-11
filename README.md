# SpoofDPI
SpoofDPI is a simple proxy tool, written in Go. It is designed to neutralize the Deep Packet Inspection (DPI) techniques that power many internet censorship systems. By operating at the TCP level, it employs packet manipulation strategies to evade SNI-based filtering, effectively granting users access to restricted content.

```txt
 ❯ spoofdpi

 .d8888b.                              .d888 8888888b.  8888888b. 8888888
d88P  Y88b                            d88P'  888  'Y88b 888   Y88b  888
Y88b.                                 888    888    888 888    888  888
 'Y888b.   88888b.   .d88b.   .d88b.  888888 888    888 888   d88P  888
    'Y88b. 888 '88b d88''88b d88''88b 888    888    888 8888888P'   888
      '888 888  888 888  888 888  888 888    888    888 888         888
Y88b  d88P 888 d88P Y88..88P Y88..88P 888    888  .d88P 888         888
 'Y8888P'  88888P'   'Y88P'   'Y88P'  888    8888888P'  888       8888888
           888
           888
           888

Press 'CTRL + c' to quit
```

<a href="https://repology.org/project/spoofdpi/versions">
    <img src="https://repology.org/badge/vertical-allrepos/spoofdpi.svg?columns=1" alt="Packaging status">
</a>

# Dependencies
```
go >= 1.22
libpcap
```

# Installation
## Using go install
```sh
# Installs the 'spoofdpi' executable to $GOBIN (or $GOPATH/bin if $GOBIN is not set)
go install github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest

# To specify a custom directory
GOBIN=~/.local/bin go install github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest
```

## Build from the source
```sh
CGO_ENABLED=1 go build -ldflags="-w -s" ./cmd/...
```

# Usage
```
Usage: spoofdpi [options...]
DESCRIPTION:
  spoofdpi - Simple and fast anti-censorship tool to bypass DPI
COPYRIGHT:
  Apache License, Version 2.0, January 2004
USAGE:
  spoofdpi [global options]
GLOBAL OPTIONS:
  --allow string
        perform DPI circumvention only on domains matching this regex pattern;
        can be given multiple times. these values have have higher priority
        than the values given with '--ignore' flag
  --cache-shards int
        number of shards to use for ttlcache; it is recommended to set this to be
        at least the number of CPU cores for optimal performance (default: 32, max: 255)
  --clean bool
        if set, all configuration files will be ignored
  --config, -c string
        custom location of the config file to load. options given through the command
        line flags will override the options set in this file. when not given, it will
        search sequentially in the following locations:
        $SPOOFDPI_CONFIG, /etc/spoofdpi.toml, $XDG_CONFIG_HOME/spoofdpi/spoofdpi.toml
        and $HOME/.config/spoofdpi/spoofdpi.toml
  --dns-addr string
        dns address (default: 8.8.8.8)
  --dns-ipv4-only bool
        resolve only IPv4 addresses
  --dns-port int
        port number for dns (default: 53)
  --doh-endpoint string
        endpoint for 'dns over https' (default: "https://${DNS_ADDR}/dns-query")
  --enable-doh bool
        enable 'dns-over-https'
  --fake-https-packets int
        number of fake packets to send before the client hello, higher values
        may increase success, but the lowest possible value is recommended.
        try this if tcp-level fragmentation (via --window-size) does not
        work. this feature requires root privilege and the 'libpcap'
        dependency (default: 0, max: 255)
  --ignore string
        do not perform DPI circumvention on domains matching this regex
        pattern; can be given multiple times.
  --listen-addr string
        IP address to listen on (default: 127.0.0.1)
  --listen-port int
        port number to listen on (default: 8080)
  --log-level string
        set log level (default: 'info')
  --silent bool
        do not show the banner and server information at start up
  --system-proxy bool
        automatically set system-wide proxy configuration
  --timeout int
        timeout in milliseconds; no effect when the value is 0 (default: 0, max: 66535)
  --version, -v bool
        print version; this may contain some other relevant information
  --window-size int
        chunk size, in number of bytes, for fragmented client hello,
        try lower values if the default value doesn't bypass the DPI;
        when not given, the client hello packet will be sent in two parts:
        fragmentation for the first data packet and the rest (default: 0, max: 255)
  --help, -h bool
        show help
```
> If you are using any vpn extensions such as Hotspot Shield in Chrome browser,
  go to Settings > Extensions, and disable them.

## OSX
Run `spoofdpi` with `--system-proxy` flag, and it will automatically setup your system proxy configuration.

## Linux
SpoofDPI does not support automatic system-wide proxy setup on Linux. Therefore, you must set the system proxy manually or run programs with the necessary environment variables or proxy options as shown below.

```sh
# Google Chrome
google-chrome --proxy-server="http://127.0.0.1:8080"

# Discord
env \
  HTTP_PROXY=127.0.0.1:8080 \
  HTTPS_PROXY=127.0.0.1:8080 \
  discord

# ...
```

# ⚙️ Configuration

SpoofDPI also supports configuration via a `TOML` file. The configuration file is searched for in the following order of precedence:

1.  The path specified by the `--config` (or `-c`) flag, or the `SPOOFDPI_CONFIG` environment variable.
2.  `/etc/spoofdpi.toml`
3.  `$XDG_CONFIG_HOME/spoofdpi/spoofdpi.toml`
4.  `$HOME/.config/spoofdpi/spoofdpi.toml`

> **Note:** An example configuration file can be found at [**example\_config.toml**](https://github.com/xvzc/SpoofDPI/blob/main/example_config.toml).

# How it works
### HTTP
Given that most websites now support HTTPS, SpoofDPI does not implement Deep Packet Inspection bypass for HTTP requests. However, it still proxies all HTTP traffic.

### HTTPS
HTTPS encrypts your data, but the very first packet (the Client Hello) still sends the website name you're visiting (the SNI) in plain text. This lets any network spy (DPI) easily see and block where you are going.

SpoofDPI stops this by messing with the connection before the DPI can read the SNI. It breaks the Client Hello into tiny fragments and injects junk packets at the TCP level, confusing the inspection and letting your connection slip through.

# Inspirations
[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel) by @SadeghHayeri  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) by @ValdikSS
