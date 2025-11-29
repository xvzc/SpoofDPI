# fake-count

`type: uint8`

## Description

Specifies the number of fake HTTPS packets to be sent before the requests `(default: 0)`

Setting this option might be helpful when the ISP of your location reassembles fragmented packets. It will send pre-constructed Client Hello that seem to be normal before the real TLS Handshake process begins. Although this feature is very powerful, it is still recommended to set the value of `window-size` to some low value (e.g. `25`) in order to achieve the successful TCP connection to the destination address.

!!! note
    SpoofDPI must be run as root privilege to use this option.

## Usage

### Command-Line Flag
```console
$ sudo spoofdpi --fake-count 7
```

### TOML Config
```toml
fake-count = 7
```
