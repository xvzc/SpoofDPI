# Configuration Examples

This page provides various configuration examples to help you set up SpoofDPI for different scenarios.

## Basic Setup

A minimal configuration to get started with DNS over HTTPS (DoH) and basic DPI bypass.

```toml
[dns]
    mode = "https"
    https-url = "https://dns.google/dns-query"
```

## Aggressive Bypass

If the default settings are not enough, you can try more aggressive settings. This configuration uses multiple fake packets and disorders the Client Hello.

```toml
[https]
    fake-count = 5
    fake-packet = [0x16, 0x03, 0x01] # Simple fake Client Hello prefix
    disorder = true
    split-mode = "chunk"
    chunk-size = 1
```

## Rule-Based Routing

Route traffic differently based on the domain or IP address.

```toml
[policy]
    # Block ads
    [[policy.overrides]]
        name = "block ads"
        match = { domain = "ads.example.com" }
        block = true

    # Bypass DPI for specific blocked site
    [[policy.overrides]]
        name = "unblock site"
        match = { domain = "blocked-site.com" }
        https = { fake-count = 7, disorder = true }

    # Use local network directly (no processing)
    [[policy.overrides]]
        name = "local bypass"
        match = { cidr = "192.168.0.0/16", port = "all" }
        https = { skip = true }
```

## Automatic Detection

Let SpoofDPI automatically detect blocked sites and apply a specific template.

```toml
[policy]
    auto = true

    # Template applied to automatically detected blocked sites
    [policy.template]
        https = { fake-count = 7, disorder = true }
```
