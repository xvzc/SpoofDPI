# Server Configuration

Settings related to the proxy server connection and listener.

## `listen-addr`

`type: <ip:port>`

### Description

Specifies the IP address and port to listen on. `(default: 127.0.0.1:8080)`

If you want to run SpoofDPI remotely (e.g., on a physically separated machine), set the IP part to `0.0.0.0`. Otherwise, it is recommended to leave this option as default for security.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --listen-addr "0.0.0.0:8080"
```

**TOML Config**
```toml
[server]
listen-addr = "0.0.0.0:8080"
```

---

## `timeout`

`type: uint16`

### Description

Specifies the timeout (in milliseconds) for every TCP connection. `(default: 0, max: 65535)`

A value of `0` means no timeout. You can set this option if you know what you are doing, but in most cases, leaving this option unset is recommended.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --timeout 5000
```

**TOML Config**
```toml
[server]
timeout = 5000
```

---

## `default-ttl`

`type: uint8`

### Description

Specifies the default [Time To Live (TTL)](https://en.wikipedia.org/wiki/Time_to_live) value for outgoing packets. `(default: 64)`

This value is used to restore the TTL to its default state after applying disorder strategies. Changing this option is generally not required.

!!! note
    The default TTL value for macOS and Linux is usually `64`.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --default-ttl 128
```

**TOML Config**
```toml
[server]
default-ttl = 128
```
