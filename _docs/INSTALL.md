# Installation

## Homebrew 🍻
```bash
brew install spoofdpi
```
### Run with binary
```bash
spoofdpi
```
### Run as a service
```bash
brew services start spoofdpi # This will automatically relaunch SpoofDPI on startup
```

```bash
brew services run spoofdpi # This will not relaunch SpoofDPI on startup
```
### Logs
If you run SpoofDPI as a service, it will write logs into the files below
```bash
$HOMEBREW_PREFIX/var/log/spoofdpi/output.log
$HOMEBREW_PREFIX/var/log/spoofdpi/error.log
```

## Curl
SpoofDPI will be installed in `~/.spoofdpi/bin`.
To run SpoofDPI in any directory, add the line below to your `~/.bashrc || ~/.zshrc || ...`
```bash
export PATH=$PATH:~/.spoofdpi/bin
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

# Run
```bash
spoofdpi
```

## Go
You can also install SpoofDPI with `go install`
```bash
go install github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest
```

## Git
You can also build your own
```bash
git clone https://github.com/xvzc/SpoofDPI.git
cd SpoofDPI
go build ./cmd/...
```

