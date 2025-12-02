# https-disorder

`type: boolean`

## Description

Specifies whether to disorder fragmented Client Hello packets. `(default: false)`

When enabled, this option varies the TTL of fragmented Client Hello packets. 
This simulates network disorder, potentially causing fragments to arrive out of order. 
It complicates the packet reassembly process, improving bypass reliability.

!!! note
    If [https-chunk-size](./https-chunk-size.md) is `0`, all data is sent as a single chunk, so disordering does not occur.

## Usage

### Command-Line Flag
```console
$ spoofdpi --https-disorder
```

### TOML Config
```toml
https-disorder = true
```
