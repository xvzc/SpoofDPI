# fake-https-packets

`type: uint8`

## Description

Specifies the number of fake HTTPS packets to be sent before the requests `(default: 0)`

Setting `fake-https-packets` might be helpful when the ISP of your location reassembles fragmented Client Hello packet, It will send pre-constructed HTTPS packets that seem to be normal before the real TLS Handshake process begins. Although this feature is very powerful, it is still recommended to set the value of `window-size` to some low value (e.g. `window-size=1`) in order to achieve the successful TCP connection to the destination address.

!!! note
    SpoofDPI must be run as root privilage to use this option.

## Usage

### Command-Line Flag
```console
$ sudo spoofdpi --fake-https-packets 10
```

### TOML Config
```toml
fake-https-packets = 10
```
