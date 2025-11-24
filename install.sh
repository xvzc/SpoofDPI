#!/bin/bash
set -e

# Configuration
OWNER="xvzc"
REPO="SpoofDPI"
BIN_NAME="spoofdpi"
INSTALL_PATH="/usr/local/bin"

# Helper function to run commands with sudo if needed
run_priv() {
  if [ -w "${INSTALL_PATH}" ]; then
    "$@"
  else
    sudo "$@"
  fi
}

# Detect Operating System
OS="$(uname -s)"
case "${OS}" in
Linux*) OS_TYPE="linux" ;;
Darwin*) OS_TYPE="darwin" ;;
*)
  echo "Unsupported operating system: ${OS}"
  exit 1
  ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
x86_64 | amd64) ARCH_TYPE="x86_64" ;;
arm64 | aarch64) ARCH_TYPE="arm64" ;;
armv7* | armv6*) ARCH_TYPE="arm" ;;
i386 | i686) ARCH_TYPE="i386" ;;
riscv64) ARCH_TYPE="riscv64" ;;
mips*) ARCH_TYPE="mips" ;;
*)
  echo "Unsupported architecture: ${ARCH}"
  exit 1
  ;;
esac

# Resolve latest version tag from GitHub API
echo "Resolving latest version..."
LATEST_URL="https://api.github.com/repos/${OWNER}/${REPO}/releases/latest"
TAG=$(curl -s "${LATEST_URL}" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "${TAG}" ]; then
  echo "Failed to fetch latest version tag."
  exit 1
fi

# Remove 'v' prefix for filename construction
VERSION="${TAG#v}"

# Construct filename
FILE_NAME="${BIN_NAME}_${VERSION}_${OS_TYPE}_${ARCH_TYPE}.tar.gz"
DOWNLOAD_URL="https://github.com/${OWNER}/${REPO}/releases/download/${TAG}/${FILE_NAME}"

# Prepare temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "Downloading ${FILE_NAME}..."
curl -sL -o "${TMP_DIR}/${FILE_NAME}" "${DOWNLOAD_URL}"

if [ ! -f "${TMP_DIR}/${FILE_NAME}" ]; then
  echo "Download failed. Please check your architecture compatibility or network connection."
  exit 1
fi

# Extract binary
echo "Extracting..."
tar -xzf "${TMP_DIR}/${FILE_NAME}" -C "${TMP_DIR}"

# Install binary
echo "Installing to ${INSTALL_PATH}..."
if [ ! -w "${INSTALL_PATH}" ]; then
  echo "Admin permission required to install to ${INSTALL_PATH}"
fi

# Move binary with permission check
run_priv mv "${TMP_DIR}/${BIN_NAME}" "${INSTALL_PATH}/${BIN_NAME}"

# Set executable permissions with permission check
run_priv chmod +x "${INSTALL_PATH}/${BIN_NAME}"

echo "Successfully installed ${BIN_NAME} ${TAG} to ${INSTALL_PATH}/${BIN_NAME}"
