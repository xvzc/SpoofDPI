## Requirements

SpoofDPI requires the [libpcap](https://github.com/the-tcpdump-group/libpcap) library on all operating systems **except Linux**.

```console
- MacOS
$ brew install libpcap 

- FreeBSD
$ pkg install libpcap

- ...
```

## Install Using curl
You can install `spoofdpi` using `curl`. The binary will be installed to `/usr/local/bin`.
```console
$ curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash
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

- ...
```

## Manual Build
To build SpoofDPI manually, ensure that you have a recent version of [Go](https://go.dev) and the [libpcap](https://github.com/the-tcpdump-group/libpcap) library installed.
!!! note 
    **libpcap** is no longer required on Linux, so `CGO` does not need to be enabled.
### Git
If you clone the repository to build manually, we recommend including the commit hash for better issue tracking.
```console
$ git clone https://github.com/xvzc/SpoofDPI
$ cd SpoofDPI
$ CGO_ENABLED=1 go build -ldflags "-s -w" \
    -ldflags "-X 'main.commit=$(git rev-parse --short HEAD 2>/dev/null)'" \
    -ldflags "-X 'main.build=manual'" \
    -o spoofdpi ./cmd/spoofdpi

```
### Github Release
You can also build from the release source code. For platforms where native GitHub Actions runners are unavailable (e.g., FreeBSD), manual packaging is required. Please set the version and build information so that maintainers can track issues easily.
```console
$ CGO_ENABLED=1 go build -ldflags "-s -w" \
    -ldflags "-X 'main.version=1.0.2'" \
    -ldflags "-X 'main.build=freebsd'" \
    -o spoofdpi ./cmd/spoofdpi
```


## Packaging Status
<a href="https://repology.org/project/spoofdpi/versions">
    <img src="https://repology.org/badge/vertical-allrepos/spoofdpi.svg?columns=1" alt="Packaging status">
</a>

