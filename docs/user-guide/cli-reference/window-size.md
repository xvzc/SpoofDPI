# window-size

`type: uint8`

## Description

Specifies the chunk size in bytes for the Client Hello packet.
Try lower values if the default fails to bypass the DPI.
Setting this to 0 disables fragmentation. `(default: 35, max: 255)`

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
