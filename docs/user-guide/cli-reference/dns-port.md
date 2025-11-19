# dns-port

`type: uint16`

## Description

Specifies the port number to send dns queries to. `(default: 53)`

Set this value if you run a self-hosted *[Domain Name Server (DNS)](https://en.wikipedia.org/wiki/Domain_Name_System)* with a custom port.

## Usage

### Command-Line Flag
```console
$ spoofdpi --dns-port 3000
```

### TOML Config
```toml
dns-port = 3000
```
