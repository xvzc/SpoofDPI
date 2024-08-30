# Quick Start

## Binary
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

## Linux

### Arch Linux

<a href="https://aur.archlinux.org/packages/spoofdpi"><img alt="AUR spoofdpi package" src="https://img.shields.io/aur/version/spoofdpi?style=flat&label=spoofdpi&logo=archlinux"></a>
<a href="https://aur.archlinux.org/packages/spoofdpi-bin"><img alt="AUR spoofdpi-bin package" src="https://img.shields.io/aur/version/spoofdpi-bin?style=flat&label=spoofdpi-bin&logo=archlinux"></a>
<a href="https://aur.archlinux.org/packages/spoofdpi-git"><img alt="AUR spoofdpi-git package" src="https://img.shields.io/aur/version/spoofdpi-git?style=flat&label=spoofdpi-git&logo=archlinux"></a>

### ALT Sisyphus

<a href="https://packages.altlinux.org/en/sisyphus/srpms/spoof-dpi/">spoof-dpi</a>

### ROSA

<a href="https://abf.io/import/spoofdpi">spoofdpi</a>

## FreeBSD 😈
```
# Build from ports tree
make -C /usr/ports/net/spoofdpi install clean
# Install the package
pkg install spoofdpi
```

## Homebrew 🍻

<a href="https://formulae.brew.sh/formula/spoofdpi"><img alt="Homebrew spoofdpi formula" src="https://img.shields.io/homebrew/v/spoofdpi?style=flat&logo=homebrew&label=spoofdpi"></a>

```bash
brew install spoofdpi
```

## Go
You can also install SpoofDPI with `go install`.
```bash
go install github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest
```

## How to build

```bash
git clone https://github.com/xvzc/SpoofDPI.git
cd SpoofDPI
go build ./cmd/...
```
