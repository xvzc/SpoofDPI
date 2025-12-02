# https-fake-count

`type: uint8`

## Description

Specifies the number of fake packets to be sent before the Client Hello. If 'https-chunk-size' is greater than 0, each fake packet will be fragmented into segments of the specified window size. `(default: 0)`

Setting this option might be helpful when the ISP of your location reassembles fragmented packets. It will send pre-constructed Client Hello that seem to be normal before the real TLS Handshake process begins. Although this feature is very powerful, it is still recommended to set the value of `https-chunk-size` to some low value (e.g. `25`) in order to achieve the successful TCP connection to the destination address.

!!! note
    SpoofDPI must be run as root privilege to use this option.

## Usage

### Command-Line Flag
```console
$ sudo spoofdpi --https-fake-count 7
```

### TOML Config
```toml
https-fake-count = 7
```