# window-size

`type: uint8`

## Description

Specifies the chunk size (in bytes) when performing TCP-level fragmentation on Client Hello packet `(default: 0)`

The fragmentation strategy works differently depending on the value of this option:

- **If the value is 0:** Uses `legacy style fragmentation`. The Client Hello is split into two parts: the first 1 byte and the rest of the packet.

- **If the value is greater than 0:** Uses `chunking strategy`. The Client Hello is split into multiple chunks, each with a maximum size of this value.

If your ISP seems to reassemble the fragmented Client Hello packet, setting this value to `1` might help increase the possibility of successful circumvention. See also [fake-https-packets](fake-https-packets.md) for a better experience.

## Usage

### Command-Line Flag
```console
$ spoofdpi --window-size 1
```

### TOML Config
```toml
window-size = 1
```
