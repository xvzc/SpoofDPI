# dns-qtype

`type: one_of['ipv4', 'ipv6', 'all']`

## Description

Filters DNS queries by record type (A for IPv4, AAAA for IPv6).
Available values are `"ipv4"`, `"ipv6"`, and `"all"`. `(default: "ipv4")`

If your *[Internet Service Provider (ISP)](https://en.wikipedia.org/wiki/Internet_service_provider)* doesn't support IPv6, it is recommended to set this option to `"ipv4"` for stability.

## Usage

### Command-Line Flag
```console
$ spoofdpi --dns-qtype "all"
```

### TOML Config
```toml
dns-qtype = "all"
```
