# General Configuration

General settings for the application, including logging and system integration.

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
[general]
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
[general]
silent = true
```

---

## `system-proxy`

`type: boolean`

### Description

Specifies whether to automatically set up the system-wide proxy configuration. `(default: false)`

!!! important
    This option is currently only supported on **macOS**.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --system-proxy
```

**TOML Config**
```toml
[general]
system-proxy = true
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
