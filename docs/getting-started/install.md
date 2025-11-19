## Requirements

SpoofDPI requires a recent version of [Go](https://go.dev) and the [libpcap](https://github.com/the-tcpdump-group/libpcap) library to be installed on your system.
You can install these dependencies using the package manager of your choice, as shown below.

```console
- MacOS
$ brew install go libpcap 

- Arch Linux
$ yay -S go libpcap

# ...
```

## Install Using Go
You can install `spoofdpi` using `go install`.
```console
- Method 1: Default Install
   Installs to your default Go bin path ($GOPATH/bin or $HOME/go/bin).
   This directory must be in your system's $PATH.
$ go install github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest

- Method 2: System-Wide Install (Recommended)
   Installs to /usr/local/bin, which is standard for system-wide binaries.
$ GOBIN=/usr/local/bin sudo go install github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest

- Method 3: User-Wide Install (Alternative)
   Installs to a user-specific local bin directory.
   Make sure $HOME/.local/bin is in your $PATH.
$ GOBIN=$HOME/.local/bin go install github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest
```

## Install With Package Manager
You can also install SpoofDPI using package managers of your choice, but note that the version may not be the latest, depending on the [Packaging Status](#packaging-status).

```console
- MacOS
$ brew install spoofdpi

- Arch Linux
$ yay -S spoofdpi

- Fedora
$ sudo dnf install spoofdpi

- FreeBSD
$ pkg install spoofdpi
```

## Packaging Status
<a href="https://repology.org/project/spoofdpi/versions">
    <img src="https://repology.org/badge/vertical-allrepos/spoofdpi.svg?columns=1" alt="Packaging status">
</a>

