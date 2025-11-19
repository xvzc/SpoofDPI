# listen-port

`type: uint16`

## Description

Specifies the port number to listen on. `(default: 8080)`

root privilege may be required if you set this value under `1024`.

## Usage

### Command-Line Flag
```console
$ spoofdpi --listen-port 8000
```

### TOML Config
```toml
listen-port = 8000
```
