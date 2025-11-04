# Installation Guide

## Table of Contents

<!--ts-->
   * [Binary](#binary)
   * [Go](#go)
   * [Snap](#snap)
<!--te-->

## Binary
SpoofDPI will be installed in `~/.spoofdpi/bin`.  
To run SpoofDPI in any directory, add the line below to your `~/.bashrc || ~/.zshrc || ...`
```bash
export PATH=$PATH:~/.spoofdpi/bin
```
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
```bash
go install github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest
```

## Snap
```bash
# Install from snapstore
snap install spoofdpi

# List services
snap services spoofdpi

# Enable and start spoofdpi.spoofdpi-daemon snap service
snap start --enable spoofdpi.spoofdpi-daemon

# Show generated systemd service status
systemctl status snap.spoofdpi.spoofdpi-daemon.service

# Override generated systemd service (configure startup options)
systemctl edit snap.spoofdpi.spoofdpi-daemon.service

## NOTE: you can pass args to spoofdpi:
##  [Service]
##  ExecStart=
##  ExecStart=/usr/bin/snap run spoofdpi.spoofdpi-daemon --arg1 --arg2 .....

# Restart generated systemd service to apply changes
systemctl restart snap.spoofdpi.spoofdpi-daemon.service

# ... and show service status
systemctl status snap.spoofdpi.spoofdpi-daemon.service
```
