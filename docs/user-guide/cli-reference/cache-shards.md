# cache-shards

`type: uint8`

## Description

Specifies the number of shards used for the cache. (default: 32)

Cached data (e.g., DNS records) must be thread-safe. To avoid lock contention, SpoofDPI stores data across multiple shards.

For optimal performance, it is recommended to set this value to be greater than or equal to the number of available CPU cores

## Usage
### Command-Line Flag
```console
$ spoofdpi --cache-shards 64
```

### TOML Config
```toml
cache-shards = 64
```
