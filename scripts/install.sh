#!/bin/bash
# Install BlackCat AI Agent CLI
# Usage: curl -fsSL https://raw.githubusercontent.com/Meow-AIs/BlackCat/main/scripts/install.sh | bash
set -euo pipefail

REPO="Meow-AIs/BlackCat"
BINARY="blackcat"

# Detect OS
detect_os() {
    local os
    os="$(uname -s)"
    case "$os" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "unsupported"; return 1 ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "unsupported"; return 1 ;;
    esac
}

# Find install directory
find_install_dir() {
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
    elif [ -d "$HOME/.local/bin" ]; then
        echo "$HOME/.local/bin"
    else
        mkdir -p "$HOME/.local/bin"
        echo "$HOME/.local/bin"
    fi
}

# Get latest release version
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
        grep '"tag_name"' |
        sed -E 's/.*"v?([^"]+)".*/\1/'
}

main() {
    echo "Installing BlackCat..."
    echo ""

    local os arch version install_dir
    os="$(detect_os)"
    arch="$(detect_arch)"

    if [ "$os" = "unsupported" ] || [ "$arch" = "unsupported" ]; then
        echo "Error: unsupported platform $(uname -s)/$(uname -m)"
        exit 1
    fi

    # Windows + arm64 not supported
    if [ "$os" = "windows" ] && [ "$arch" = "arm64" ]; then
        echo "Error: Windows ARM64 is not supported"
        exit 1
    fi

    version="$(get_latest_version)"
    if [ -z "$version" ]; then
        echo "Error: could not determine latest version"
        exit 1
    fi

    install_dir="$(find_install_dir)"

    echo "  OS:       $os"
    echo "  Arch:     $arch"
    echo "  Version:  $version"
    echo "  Install:  $install_dir"
    echo ""

    local ext="tar.gz"
    if [ "$os" = "windows" ]; then
        ext="zip"
    fi

    local filename="${BINARY}_${version}_${os}_${arch}.${ext}"
    local url="https://github.com/${REPO}/releases/download/v${version}/${filename}"

    echo "Downloading $url..."
    local tmpdir
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    curl -fsSL -o "$tmpdir/$filename" "$url"

    echo "Extracting..."
    if [ "$ext" = "tar.gz" ]; then
        tar -xzf "$tmpdir/$filename" -C "$tmpdir"
    else
        unzip -q "$tmpdir/$filename" -d "$tmpdir"
    fi

    local binary_name="$BINARY"
    if [ "$os" = "windows" ]; then
        binary_name="${BINARY}.exe"
    fi

    cp "$tmpdir/$binary_name" "$install_dir/$binary_name"
    chmod +x "$install_dir/$binary_name"

    echo ""
    echo "BlackCat v${version} installed to ${install_dir}/${binary_name}"

    # Check if install dir is in PATH
    if ! echo "$PATH" | tr ':' '\n' | grep -qx "$install_dir"; then
        echo ""
        echo "NOTE: $install_dir is not in your PATH."
        echo "Add it with: export PATH=\"\$PATH:$install_dir\""
    fi

    echo ""
    echo "Run 'blackcat init' to set up configuration."
    echo "Run 'blackcat help' for usage information."
}

main
