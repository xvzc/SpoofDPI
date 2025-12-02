# https-split-default

`type: one_of['chunk', '1byte', 'sni', 'none']`

## Description

Specifies the default packet fragmentation strategy to use. `(default: 'chunk')`

- **'chunk'**: Splits the Client Hello into segments of `https-chunk-size`.
- **'1byte'**: Sends the first byte of Client Hello separately.
- **'sni'**: Splits the Client Hello right before the SNI field.
- **'none'**: Disables fragmentation.

## Usage

### Command-Line Flag
```console
$ spoofdpi --https-split-default "sni"
```

### TOML Config
```toml
https-split-default = "sni"
```
