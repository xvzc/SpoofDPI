## Requirements

SpoofDPI requires the [libpcap](https://github.com/the-tcpdump-group/libpcap) library on all operating systems **except Linux**.

```console
- macOS
$ brew install libpcap 

- FreeBSD
$ pkg install libpcap

- Linux
$ echo "libpcap is not required on Linux"

- ...
```

## Install Using Script
You can install `spoofdpi` using the provided script. The binary will be installed to `/usr/local/bin`.
```console
$ curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash
```

## Install With Package Manager
You can also install SpoofDPI using package managers of your choice, but note that the version may not be the latest, depending on the [Packaging Status](#packaging-status).

```console
- macOS
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
If you are building manually from the latest commit, we recommend including the commit hash for better issue tracking.

```sh
#!/usr/bin/env sh

BUILD_INFO="git"
SRC="SpoofDPI"
DIST="dist"

mkdir -p ./$DIST

git clone https://github.com/xvzc/SpoofDPI.git

BUILD_LDFLAGS="-s -w"
BUILD_LDFLAGS="$BUILD_LDFLAGS -X 'main.commit=$(git -C ./$SRC rev-parse --short HEAD)'"
BUILD_LDFLAGS="$BUILD_LDFLAGS -X 'main.build=$BUILD_INFO'"

# You can disable CGO on Linux by setting `CGO_ENABLED=0`
CGO_ENABLED=1 go build -C ./$SRC \
  -ldflags "$BUILD_LDFLAGS" \
  -o ../$DIST/spoofdpi ./cmd/spoofdpi
```

### GitHub Release

You can also build directly from the release source code. This is particularly useful for platforms where native GitHub Actions runners are unavailable (e.g., FreeBSD), requiring manual packaging.

We recommend injecting version and build information during the build process to help maintainers track issues effectively.

Every release includes a custom source archive (e.g., `spoofdpi-1.1.3.tar.gz`) which contains a `COMMIT` file. You can use this file to embed the commit hash into the binary.
```bash
#!/usr/bin/env bash

VERSION="#REPLACE_THIS_WITH_VERSION#"
BUILD_INFO="freebsd"
ASSET="spoofdpi-$VERSION.tar.gz"
SRC="spoofdpi-$VERSION"
DIST="dist"

curl -fsSL \
  https://github.com/xvzc/SpoofDPI/releases/download/v$VERSION/$ASSET \
  -o ./$ASSET

tar -xvzf ./spoofdpi-$VERSION.tar.gz

BUILD_LDFLAGS="-s -w"
BUILD_LDFLAGS="$BUILD_LDFLAGS -X 'main.version=$VERSION'"
BUILD_LDFLAGS="$BUILD_LDFLAGS -X 'main.commit=$(cat ./$SRC/COMMIT)'"
BUILD_LDFLAGS="$BUILD_LDFLAGS -X 'main.build=$BUILD_INFO'"

# You can disable CGO on Linux by setting `CGO_ENABLED=0`
CGO_ENABLED=1 go build -C ./$SRC \
  -ldflags "$BUILD_LDFLAGS" \
  -o ../$DIST/spoofdpi ./cmd/spoofdpi
```


## Packaging Status
<a href="https://repology.org/project/spoofdpi/versions">
    <img src="https://repology.org/badge/vertical-allrepos/spoofdpi.svg?columns=1" alt="Packaging status">
</a>

