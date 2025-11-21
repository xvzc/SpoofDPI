# auto-policy

`type: boolean`

## Description

Enables automatic detection of blocked domains and the addition of corresponding policies.
The request might fail until certain blocked domains are detected (typically on the first attempt).

This option, which enables automatic policy addition upon failure, can be used in conjunction with the [policy](policy.md) option for comprehensive control.


## Usage
### Command-Line Flag
```console
$ spoofdpi --auto-policy
```

### TOML Config
```toml
auto-policy = true
```
