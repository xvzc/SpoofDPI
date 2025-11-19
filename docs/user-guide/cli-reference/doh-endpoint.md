# doh-endpoint

`type: string`

## Description

Specifies the endpoint used for querying IP addresses via DOH. The endpoint should be written in `https` scheme. `(default: https://$dns-addr/dns-query)`

The default value for this option can be automatically generated from the value of the `dns-addr`. For example, If you set `dns-addr` to `1.1.1.1` and leave this option empty, the endpoint for DOH will default to `https://1.1.1.1/dns-query`

## Usage

### Command-Line Flag
```console
$ spoofdpi --doh-endpoint "https://dns.google/dns-query"
```

### TOML Config
```toml
doh-endpoint = "https://dns.google/dns-query"
```
