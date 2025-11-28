# window-size

`type: uint8`

## Description

Specifies the chunk size (in bytes) when performing TCP-level fragmentation on Client Hello packet `(default: 35)`

!!! note 
    Setting this value to `0` disables fragmentation.

If your ISP seems to reassemble the fragmented Client Hello packet, setting this value to `1` might help increase the possibility of successful circumvention. See also [fake-count](fake-count.md) for a better experience.

## Usage

### Command-Line Flag
```console
$ spoofdpi --window-size 1
```

### TOML Config
```toml
window-size = 1
```
