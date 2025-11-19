# listen-addr

`type: string`

## Description

Specifies the IP address to listen on. `(default: 127.0.0.1)`

If you want to run SpoofDPI remotely (e.g., on a physically separated machine), then you should set this value to `0.0.0.0`.  Otherwise, it is recommended to leave this option as default.

## Usage

### Command-Line Flag
```console
$ spoofdpi --listen-addr "0.0.0.0"
```

### TOML Config
```toml
listen-addr = "0.0.0.0"
```
