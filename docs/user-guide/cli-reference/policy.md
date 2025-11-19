# policy

`type: list(string)`

## Description

Specifies a list of policies that determines whether to perform circumvention on matching domain names. This option can be given multiple times. **It is highly recommended to set this option** because performing circumvention on domains that are not banned (e.g. `google.com`) can cause unexpected connection failures.

- **Rule:** Each policy must be prefixed with `i:` (include) or `x:` (exclude).
- **Matching:** Rules support the wildcard character (\*) and globstar(\*\*).
- **Priority:** More specific rules have higher priority (e.g., `x:cdn.discordapp.com` has higher priority over `i:*.discordapp.com`).

!!! tip "More information on matching strategy"
    - **Wildcard (`*`)**  
      Matches exactly **one** domain part (e.g., `www`) or **zero** parts.
      For example, `*.youtube.com` matches both `www.youtube.com` and `youtube.com`.
    - **Globstar (`**`)**  
      Matches **zero or more** nested domain parts.
      For example, `**.firefox.com` matches `firefox.com`, `www.firefox.com`, 
      and `profile.accounts.firefox.com`.
    
## Usage

### Command-Line Flag
```console
$ spoofdpi --policy "i:*.discordapp.com" --policy "x:cdn.discordapp.com"
```

### TOML Config
```toml
policy = [
    "i:*.discordapp.com",
    "x:cdn.discordapp.com",
]
```
