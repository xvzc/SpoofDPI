# listen-addr

`type: <ip:port>`

## Description

Specifies the IP address and port to listen on. `(default: 127.0.0.1:8080)`

If you want to run SpoofDPI remotely (e.g., on a physically separated machine), then you should set the IP part to `0.0.0.0`. Otherwise, it is recommended to leave this option as default.

## Usage

### Command-Line Flag
```console
$ spoofdpi --listen-addr "0.0.0.0:8080"
```

### TOML Config
```toml
listen-addr = "0.0.0.0:8080"
```
