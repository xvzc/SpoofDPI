# Overview

There are two methods to configure SpoofDPI, and both can be used together.

- TOML Config File
- Command-Line Flags

## Configuration Priority
SpoofDPI applies settings in the following order:

1. Values defined in the TOML config file
2. Values passed via command-line (CLI) flags

Command-line flags will always override values set in the TOML config file.
For example, if `config.toml` contains `listen-addr = "127.0.0.1:8080"`, but the user runs `$ spoofdpi --listen-addr :9090`, the `:9090` port will be used.

## Config File Path

If a specific path is not provided via a `--config` flag, SpoofDPI will search for `spoofdpi.toml` in the following locations in order, applying only the first file found:

- `$SPOOFDPI_CONFIG` environment variable
- `/etc/spoofdpi.toml`
- `$XDG_CONFIG_HOME/spoofdpi/spoofdpi.toml`
- `$HOME/.config/spoofdpi/spoofdpi.toml`

## Options

The configuration is organized into five main categories. Click on each category to view detailed options.

| Category | Description |
| :--- | :--- |
| **[General](general.md)** | General application options (logging, system proxy, etc.). |
| **[Server](server.md)** | Server connection options (address, timeout). |
| **[DNS](dns.md)** | DNS resolution options. |
| **[HTTPS](https.md)** | HTTPS/TLS packet manipulation options. |
| **[Policy](policy.md)** | Rule-based routing and automatic bypass policies. |

## Example

The following two methods will achieve the exact same configuration.

### Method 1: Using Command-Line Flags

All settings are passed directly via flags.

```console
$ spoofdpi --dns-addr "1.1.1.1:53" --dns-https-url "https://dns.google/dns-query" --dns-mode "https"
```

### Method 2: Using a TOML Config File

Place the settings in your `spoofdpi.toml` file:

```toml
[dns]
    addr = "1.1.1.1:53"
    https-url = "https://dns.google/dns-query"
    mode = "https"
```

Then, run spoofdpi without those flags (it will automatically load the file if placed in a standard path):

```console
$ spoofdpi
```
