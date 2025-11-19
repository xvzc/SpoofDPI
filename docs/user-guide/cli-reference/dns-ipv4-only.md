# dns-ipv4-only

`type: boolean`

## Description

Sets the condition for the type of IP addresses retrieved from dns server. `(default: false)`

If your *[Internet Service Provider (ISP)](https://en.wikipedia.org/wiki/Internet_service_provider)* doesn't support IPv6, it is recommended to set this option to `true` for stability.

## Usage

### Command-Line Flag
```console
$ spoofdpi --dns-ipv4-only
```

### TOML Config
```toml
dns-ipv4-only = true
```
