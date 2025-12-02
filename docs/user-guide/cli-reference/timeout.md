# timeout

`type: uint16`

## Description

Specifies the timeout (in milliseconds) for every TCP connection.  
`(default: none, max: 66536)`

You can set this option if you know what you are doing, but in most cases, leaving this option unset will not cause any problems.

## Usage

### Command-Line Flag
```console
$ spoofdpi --timeout 5000
```

### TOML Flag
```toml
timeout = 5000
```
