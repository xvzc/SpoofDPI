# How to Configure SpoofDPI

There are two methods to configure SpoofDPI, and both can be used together.

- TOML Config File
- Command-Line Flags


## Configuration Priority
SpoofDPI applies settings in the following order

- Values defined in the TOML config file
- Values passed via command-line (CLI) flags

Command-line flags will always override values set in the TOML config file.
For example, if config.toml contains `listen-port = 5000`, but the user runs `$ spoofdpi --listen-port 8080`, the `8080` port will be used.


## Config File Path

If a specific path is not provided via a `--config` flag, SpoofDPI will search for `spoofdpi.toml` in the following locations in order, applying only the first file found:

- `$SPOOFDPI_CONFIG` environment variable
- /etc/spoofdpi.toml
- `$XDG_CONFIG_HOME/spoofdpi/spoofdpi.toml`
- `$HOME/.config/spoofdpi/spoofdpi.toml`


## Example

The following two methods will achieve the exact same configuration.

### Method 1: Using Command-Line Flags

All settings are passed directly via flags.

```console
$ spoofdpi --dns-addr "1.1.1.1" --doh-endpoint "https://dns.google/dns-query"
```

### Method 2: Using a TOML Config File

First, place the settings in your config file:

```toml
dns-addr = "1.1.1.1"
doh-endpoint = "https://dns.google/dns-query"
```
Then, run spoofdpi without those flags (it will automatically load the file):

```console
$ spoofdpi
```

