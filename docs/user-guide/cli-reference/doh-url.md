# doh-url

`type: <https_url>`

## Description

Specifies the endpoint URL for DNS over HTTPS (DoH) queries.  
`(default: "https://dns.google/dns-query")`

## Usage

### Command-Line Flag
```console
$ spoofdpi --doh-url "https://cloudflare-dns.com/dns-query"
```

### TOML Config
```toml
doh-url = "https://cloudflare-dns.com/dns-query"
```
