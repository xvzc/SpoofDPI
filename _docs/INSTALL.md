# Installation Guide
<!--ts-->
   * [Binary](#binary)
   * [Go](#go)
   * [Package Manager](#package-manager)
      * [brew](#brew)
      * [pkg](#pkg)
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

## Package Manager

### brew
```bash
brew install spoofdpi
```

### pkg
```bash
pkg install spoofdpi
```

### nix 
You can either:
1. Add it inside NixOS configuartion file:
```Nix
environment.systemPackages = [ pkgs.spoofdpi ];
```
2. Add it to home-manager configuration:
```Nix
home.packages = [ pkgs.spoofdpi ];
```
3. Add it to nix-darwin configuration:
```Nix
environment.systemPackages = [ pkgs.spoofdpi ];
```
4. Or add it with nix-darwin brew option:
```Nix
homebrew.brews = [ "spoofdpi" ];
```

