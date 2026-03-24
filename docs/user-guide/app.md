# App Configuration

Application-level settings including mode, logging, and system integration.

## `app-mode`

`type: string`

### Description

Specifies the proxy mode. `(default: "http")`

### Allowed Values

- `http`: HTTP proxy mode
- `socks5`: SOCKS5 proxy mode
- `tun`: TUN interface mode (transparent proxy)

!!! warning
    **SOCKS5** and **TUN** modes are currently Experimental. You might encounter unexpected behaviors or bugs. Feedback and bug reports are highly appreciated!

### Usage

**Command-Line Flag**
```console
$ spoofdpi --app-mode socks5
```

**TOML Config**
```toml
[app]
mode = "socks5"
```

---

## `listen-addr`

`type: <ip:port>`

### Description

Specifies the IP address and port to listen on. `(default: 127.0.0.1:8080 for http, 127.0.0.1:1080 for socks5)`

If you want to run SpoofDPI remotely (e.g., on a physically separated machine), set the IP part to `0.0.0.0`. Otherwise, it is recommended to leave this option as default for security.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --listen-addr "0.0.0.0:8080"
```

**TOML Config**
```toml
[app]
listen-addr = "0.0.0.0:8080"
```

---

## `log-level`

`type: string`

### Description

Specifies the logging verbosity.

Available values are `debug`, `trace`, `info`, `warn`, `error`, and `disabled`. `(default: "info")`

### Usage

**Command-Line Flag**
```console
$ spoofdpi --log-level trace
```

**TOML Config**
```toml
[app]
log-level = "trace"
```

---

## `silent`

`type: boolean`

### Description

Suppresses the ASCII art banner at startup. `(default: false)`

### Usage

**Command-Line Flag**
```console
$ spoofdpi --silent
```

**TOML Config**
```toml
[app]
silent = true
```

---

## `auto-configure-network`

`type: boolean`

### Description

Specifies whether to automatically set up the system-wide proxy configuration. `(default: false)`

!!! important
    This option is currently only supported on **macOS**.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --auto-configure-network
```

**TOML Config**
```toml
[app]
auto-configure-network = true
```

---

## `config`

`type: string`

### Description

Specifies the path to a custom `TOML` config file. `(default: none)`

If this option is set, SpoofDPI will not search the default directories.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --config ~/spoofdpi.toml
```

**TOML Config**
```toml
# This option is not available in TOML config
```

---

## `clean`

`type: boolean`

### Description

Specifies whether to ignore all configuration files and run with default settings. `(default: false)`

### Usage

**Command-Line Flag**
```console
$ spoofdpi --clean
```

**TOML Config**
```toml
# This option is not available in TOML config
```

---

## `version`

`type: boolean`

### Description

Prints the version string and exits. `(default: false)`

### Usage

**Command-Line Flag**
```console
$ spoofdpi --version
```

**TOML Config**
```toml
# This option is not available in TOML config
```
