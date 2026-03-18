# Connection Configuration

Settings for network connection timeouts and packet configuration.

## `dns-timeout`

`type: uint16`

### Description

Specifies the timeout (in milliseconds) for DNS connections. `(default: 5000, max: 65535)`

A value of `0` means no timeout.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --dns-timeout 3000
```

**TOML Config**
```toml
[connection]
dns-timeout = 3000
```

---

## `tcp-timeout`

`type: uint16`

### Description

Specifies the timeout (in milliseconds) for TCP connections. `(default: 10000, max: 65535)`

A value of `0` means no timeout.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --tcp-timeout 5000
```

**TOML Config**
```toml
[connection]
tcp-timeout = 5000
```

---

## `udp-idle-timeout`

`type: uint16`

### Description

Specifies the idle timeout (in milliseconds) for UDP connections. `(default: 25000, max: 65535)`

The connection will be closed if there is no read/write activity for this duration. Each read or write operation resets the timeout.

A value of `0` means no timeout.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --udp-idle-timeout 30000
```

**TOML Config**
```toml
[connection]
udp-idle-timeout = 30000
```

---

## `default-fake-ttl`

`type: uint8`

### Description

Specifies the default [Time To Live (TTL)](https://en.wikipedia.org/wiki/Time_to_live) value for fake packets. `(default: 8)`

This value is used for fake packets sent during disorder strategies. A lower value ensures fake packets expire before reaching the destination, while the real packets arrive successfully.

!!! note
    The fake TTL should be less than the number of hops to the destination.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --default-fake-ttl 10
```

**TOML Config**
```toml
[connection]
default-fake-ttl = 10
```
