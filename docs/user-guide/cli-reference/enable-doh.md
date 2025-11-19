# enable-doh

`type: boolean`

## Description

Specifies whether to enable *[DNS Over HTTPS (DOH)](https://en.wikipedia.org/wiki/DNS_over_HTTPS)*. `(default: false)`

Set this value to `true` if your ISP performs DNS based filtering.

## Usage

### Command-Line Flag
```console
$ spoofdpi --enable-doh
```

### TOML Config
```toml
enable-doh = true
```
