# default-ttl

`type: uint8`

## Description

Specifies the default [Time To Live (TTL)](https://en.wikipedia.org/wiki/Time_to_live) value for outgoing packets `(default: 64)`.

This value is used to restore the TTL to its default state after applying the disorder strategy. Changing this option is generally not required.

!!! note
    The default TTL value for macOS and Linux is usually `64`.

## Usage

### Command-Line Flag
```console
$ spoofdpi --default-ttl 128
```

### TOML Config
```toml
default-ttl = 128
```
