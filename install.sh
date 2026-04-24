#!/bin/bash
# Freelo CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/freeloio/freelo-cli/main/install.sh | bash
#
# SECURITY: Entire script is wrapped in a block to prevent partial-download execution.
{
set -euo pipefail

REPO="freeloio/freelo-cli"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="freelo"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

info() { echo -e "${GREEN}==>${NC} $1"; }
warn() { echo -e "${YELLOW}==>${NC} $1"; }
error() { echo -e "${RED}Error:${NC} $1" >&2; exit 1; }

detect_os() {
    case "$(uname -s)" in
        Darwin*) echo "darwin" ;;
        Linux*)  echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) error "Unsupported OS: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
}

get_latest_version() {
    local url="https://api.github.com/repos/${REPO}/releases/latest"
    local version
    if command -v curl &>/dev/null; then
        version=$(curl -fsSL "$url" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')
    elif command -v wget &>/dev/null; then
        version=$(wget -qO- "$url" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')
    else
        error "curl or wget is required"
    fi

    # Validate version format (prevent injection via crafted API responses)
    if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
        error "Invalid version format: '$version'"
    fi
    echo "$version"
}

verify_checksum() {
    local file="$1"
    local checksums_file="$2"

    local expected
    expected=$(grep "$(basename "$file")" "$checksums_file" | awk '{print $1}')
    if [ -z "$expected" ]; then
        warn "No checksum found for $(basename "$file") — skipping verification"
        return 0
    fi

    local actual
    if command -v sha256sum &>/dev/null; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &>/dev/null; then
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        warn "sha256sum/shasum not found — skipping checksum verification"
        return 0
    fi

    if [ "$expected" != "$actual" ]; then
        error "Checksum mismatch! Expected: $expected, Got: $actual. The download may be corrupted or tampered with."
    fi
    info "Checksum verified"
}

main() {
    info "Installing Freelo CLI..."

    local os=$(detect_os)
    local arch=$(detect_arch)
    local version=$(get_latest_version)

    if [ -z "$version" ]; then
        error "Could not determine latest version. Is the repo public and has releases?"
    fi

    info "Latest version: v${version}"
    info "Platform: ${os}/${arch}"

    local ext="tar.gz"
    [ "$os" = "windows" ] && ext="zip"

    local filename="freelo_${version}_${os}_${arch}.${ext}"
    local base_url="https://github.com/${REPO}/releases/download/v${version}"

    # Download archive + checksums
    local tmpdir
    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    info "Downloading ${filename}..."
    if command -v curl &>/dev/null; then
        curl -fsSL -o "${tmpdir}/${filename}" "${base_url}/${filename}"
        curl -fsSL -o "${tmpdir}/checksums.txt" "${base_url}/checksums.txt" || true
    else
        wget -q -O "${tmpdir}/${filename}" "${base_url}/${filename}"
        wget -q -O "${tmpdir}/checksums.txt" "${base_url}/checksums.txt" || true
    fi

    # Verify checksum
    if [ -f "${tmpdir}/checksums.txt" ]; then
        verify_checksum "${tmpdir}/${filename}" "${tmpdir}/checksums.txt"
    else
        warn "Checksums file not available — skipping verification"
    fi

    # Extract
    info "Extracting..."
    cd "$tmpdir"
    if [ "$ext" = "zip" ]; then
        unzip -q "$filename"
    else
        tar xzf "$filename"
    fi

    # Install
    local binary="${BINARY_NAME}"
    [ "$os" = "windows" ] && binary="${BINARY_NAME}.exe"

    if [ -w "$INSTALL_DIR" ]; then
        mv "$binary" "${INSTALL_DIR}/${binary}"
        chmod +x "${INSTALL_DIR}/${binary}"
    else
        info "Installing to ${INSTALL_DIR} requires elevated permissions."
        sudo mv "$binary" "${INSTALL_DIR}/${binary}"
        sudo chmod +x "${INSTALL_DIR}/${binary}"
    fi

    # Verify
    if command -v freelo &>/dev/null; then
        info "Freelo CLI v${version} installed successfully!"
        echo ""
        info "Next steps:"
        echo "  1. Run 'freelo auth login' to authenticate"
        echo "  2. Run 'freelo projects list' to see your projects"
        echo "  3. Run 'freelo skill install claude' to set up AI agent integration"
        echo ""
        echo "  Get your API key at: https://app.freelo.io/profil/nastaveni"
    else
        warn "Installed to ${INSTALL_DIR}/${binary}"
        warn "Make sure ${INSTALL_DIR} is in your PATH"
    fi
}

main
}
