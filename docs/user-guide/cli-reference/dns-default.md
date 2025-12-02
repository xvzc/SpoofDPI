# dns-default

`type: one_of['udp', 'doh', 'sys']`

## Description

Specifies the default resolution mode for domains that do not match any specific rule.
`(default: "udp")`

## Usage

### Command-Line Flag
```console
$ spoofdpi --dns-default "doh"
```

### TOML Config
```toml
dns-default = "doh"
```
