# https-chunk-size

`type: uint8`

## Description

Specifies the chunk size in bytes for the Client Hello packet.
Try lower values if the default fails to bypass the DPI.
Setting this to 0 disables fragmentation. `(default: 35, max: 255)`

If your ISP seems to reassemble the fragmented Client Hello packet, setting this value to `1` might help increase the possibility of successful circumvention. See also [https-fake-count](https-fake-count.md) for a better experience.

!!! note
    Although setting this size to '0' will internally disable fragmentation 
    (as a safeguard against division-by-zero errors), it is strongly recommended 
    to explicitly set 'https-split-default' to 'none' to properly disable this feature.

## Usage

### Command-Line Flag
```console
$ spoofdpi --https-chunk-size 1
```

### TOML Config
```toml
https-chunk-size = 1
```
