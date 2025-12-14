# Policy Configuration

By defining rules within the Policy section, you can granularly control how SpoofDPI handles connections to specific domains or IP addresses. You can define per-domain bypass strategies, DNS settings, or simply block connections.

## `auto`

`type: boolean`

### Description

Automatically detect blocked sites and add them to the bypass list. `(default: false)`

When enabled, SpoofDPI attempts to detect if a connection is being blocked and temporarily applies bypass rules for that destination. These generated rules utilize the configuration defined in `[policy.template]`.

### Usage

**Command-Line Flag**
```console
$ spoofdpi --policy-auto
```

**TOML Config**
```toml
[policy]
auto = true
```

---

## `template`

The `[policy.template]` section defines the default behavior for rules automatically generated when `auto = true`. If you enable automatic detection, you should configure this template to ensure the generated rules effectively bypass the DPI.

!!! note
    The template configuration is only available via the TOML config file.

### Structure

The template uses the same `Rule` structure as overrides, but typically only the `https` and `dns` sections are relevant, as the `match` criteria are determined dynamically.

### Example

```toml
[policy]
    auto = true

    # This configuration is applied to automatically detected blocked sites
    [policy.template]
        https = { fake-count = 7, disorder = true }
```

---

## `overrides`

Detailed policy rules are defined in the `[policy]` section of the TOML configuration file.

!!! note
    These advanced rules are only available via the TOML config file and cannot be set via command-line flags.

### Structure

The `[policy]` section contains an array of `overrides` tables. Each override rule consists of matching criteria (`match`) and specific settings for DNS (`dns`) and HTTPS (`https`).

### Rule Fields

| Field      | Type   | Description                                      |
| :--------- | :----- | :----------------------------------------------- |
| `name`     | String | A descriptive name for the rule.                 |
| `priority` | Int    | Order of precedence. Higher numbers take priority.|
| `block`    | Bool   | If `true`, completely blocks connections matching this rule. |

### Match Criteria (`match`)

You must specify either a `domain` or a `cidr` (with `port`).

| Field    | Type   | Description                                                                 |
| :------- | :----- | :-------------------------------------------------------------------------- |
| `domain` | String | Domain pattern. Supports wildcards (`*`, `**`).                             |
| `cidr`   | String | IP range in CIDR notation (e.g., `192.168.0.0/24`). Requires `port` to be set. |
| `port`   | String | Port or port range (e.g., `80`, `80-443`, `all`). Required if `cidr` is used. |

### DNS Override (`dns`)

Customize how domain names are resolved for matched traffic. The available fields mirror the global [DNS Configuration](dns.md).

| Field       | Type   | Description                                      |
| :---------- | :----- | :----------------------------------------------- |
| `mode`      | String | Resolver to use: `"udp"`, `"https"` (DoH), or `"system"`. |
| `addr`      | String | Custom upstream server (e.g., `8.8.8.8:53`).     |
| `https-url` | String | Custom DoH URL (e.g., `https://dns.google/dns-query`). |
| `qtype`     | String | Query type: `"ipv4"`, `"ipv6"`, or `"all"`.      |
| `cache`     | Bool   | If `true`, enables caching for this rule.        |

### HTTPS Override (`https`)

Customize how HTTPS connections are established. The available fields mirror the global [HTTPS Configuration](https.md).

| Field         | Type   | Description                                           |
| :------------ | :----- | :---------------------------------------------------- |
| `disorder`    | Bool   | Send Client Hello packets out of order.               |
| `fake-count`  | Int    | Number of fake packets to send.                       |
| `fake-packet` | Array  | List of bytes for the fake packet (e.g., `[0x16]`).   |
| `split-mode`  | String | Split strategy: `"chunk"`, `"sni"`, `"random"`, etc.  |
| `chunk-size`  | Int    | Size of chunks when `split-mode` is `"chunk"`.        |
| `skip`        | Bool   | If `true`, bypasses DPI modifications (standard TLS). |

### Example

```toml
[policy]
    # Example A: Allow YouTube with specific DPI bypass settings
    [[policy.overrides]]
        name = "allow youtube"
        priority = 50
        match = { domain = "*.youtube.com" }
        https = { disorder = true, fake-count = 7 }

    # Example B: Bypass DPI for local network traffic (Standard Connection)
    [[policy.overrides]]
        name = "skip local"
        priority = 51
        match = { cidr = "192.168.0.0/24", port = "all" }
        https = { skip = true }

    # Example C: Block a specific domain
    [[policy.overrides]]
        name = "block ads"
        priority = 100
        match = { domain = "ads.example.com" }
        block = true
```
