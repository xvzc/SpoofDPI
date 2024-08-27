# Installation

## Homebrew 🍻
```bash
brew install spoofdpi
```
### Usage
#### Binary
You can start SpoofDPI with:
```bash
spoofdpi
```
#### Service
Alternatively you can start SpoofDPI as a service
- with automatic relaunch on startup:
    ```bash
    brew services start spoofdpi
    ```
- without:
    ```bash
    brew services run spoofdpi
    ```
##### Logs
When running as a service, SpoofDPI writes logs to
```text
$HOMEBREW_PREFIX/var/log/spoofdpi/output.log
$HOMEBREW_PREFIX/var/log/spoofdpi/error.log
```

## Binary
SpoofDPI will be installed in `~/.spoof-dpi/bin`.
To run SpoofDPI in any directory, add the line below to your `~/.bashrc || ~/.zshrc || ...`
```
export PATH=$PATH:~/.spoof-dpi/bin
```
---
```bash
# macOS Intel
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s darwin-amd64

# macOS Apple Silicon
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s darwin-arm64

# linux-amd64
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-amd64

# linux-arm
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-arm

# linux-arm64
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-arm64

# linux-mips
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-mips

# linux-mipsle
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-mipsle
```

## Go
You can also install SpoofDPI with `go install`
```bash
$ go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi@latest
```

## Git
You can also build your own
```bash
$ git clone https://github.com/xvzc/SpoofDPI.git
$ cd SpoofDPI
$ go build ./cmd/...
```

