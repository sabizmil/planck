#!/bin/sh
# Planck installer script
# Usage: curl -sSfL https://raw.githubusercontent.com/sabizmil/planck/main/scripts/install.sh | sh
set -e

REPO="sabizmil/planck"
BINARY_NAME="planck"

# Colors (disabled if not a terminal)
if [ -t 1 ]; then
    BOLD='\033[1m'
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[0;33m'
    RESET='\033[0m'
else
    BOLD='' GREEN='' RED='' YELLOW='' RESET=''
fi

info()  { printf "${GREEN}==>${RESET} ${BOLD}%s${RESET}\n" "$1"; }
warn()  { printf "${YELLOW}warning:${RESET} %s\n" "$1"; }
error() { printf "${RED}error:${RESET} %s\n" "$1" >&2; exit 1; }

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       error "Unsupported operating system: $(uname -s). Planck supports macOS and Linux." ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        arm64|aarch64)  echo "arm64" ;;
        *)              error "Unsupported architecture: $(uname -m). Planck supports amd64 and arm64." ;;
    esac
}

# Find a download tool
detect_downloader() {
    if command -v curl >/dev/null 2>&1; then
        echo "curl"
    elif command -v wget >/dev/null 2>&1; then
        echo "wget"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# Download a URL to a file
download() {
    url="$1"
    dest="$2"
    case "$DOWNLOADER" in
        curl) curl -sSfL -o "$dest" "$url" ;;
        wget) wget -q -O "$dest" "$url" ;;
    esac
}

# Get the latest release version from GitHub
get_latest_version() {
    url="https://api.github.com/repos/${REPO}/releases/latest"
    case "$DOWNLOADER" in
        curl) version=$(curl -sSfL "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/') ;;
        wget) version=$(wget -qO- "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/') ;;
    esac

    if [ -z "$version" ]; then
        error "Could not determine the latest version. Check https://github.com/${REPO}/releases"
    fi
    echo "$version"
}

# Determine install directory
detect_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        echo "$INSTALL_DIR"
        return
    fi

    # Prefer /usr/local/bin if writable
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
        return
    fi

    # Fall back to ~/.local/bin
    local_bin="${HOME}/.local/bin"
    mkdir -p "$local_bin"
    echo "$local_bin"
}

# Verify checksum
verify_checksum() {
    archive_file="$1"
    checksums_file="$2"
    archive_name="$(basename "$archive_file")"

    expected=$(grep "$archive_name" "$checksums_file" | awk '{print $1}')
    if [ -z "$expected" ]; then
        error "No checksum found for $archive_name in checksums.txt"
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$archive_file" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$archive_file" | awk '{print $1}')
    else
        warn "Neither sha256sum nor shasum found — skipping checksum verification"
        return 0
    fi

    if [ "$actual" != "$expected" ]; then
        error "Checksum verification failed!\n  Expected: $expected\n  Got:      $actual"
    fi
}

main() {
    info "Installing Planck..."

    OS=$(detect_os)
    ARCH=$(detect_arch)
    DOWNLOADER=$(detect_downloader)

    info "Detected platform: ${OS}/${ARCH}"

    # Get version (use VERSION env var or fetch latest)
    if [ -n "$VERSION" ]; then
        version="$VERSION"
    else
        info "Fetching latest version..."
        version=$(get_latest_version)
    fi

    # Strip leading 'v' for archive name
    version_num="${version#v}"

    info "Installing Planck ${version}..."

    # Build download URLs
    archive_name="${BINARY_NAME}_${version_num}_${OS}_${ARCH}.tar.gz"
    base_url="https://github.com/${REPO}/releases/download/${version}"
    archive_url="${base_url}/${archive_name}"
    checksums_url="${base_url}/checksums.txt"

    # Create temp directory
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download archive and checksums
    info "Downloading ${archive_name}..."
    download "$archive_url" "${tmp_dir}/${archive_name}"
    download "$checksums_url" "${tmp_dir}/checksums.txt"

    # Verify checksum
    info "Verifying checksum..."
    verify_checksum "${tmp_dir}/${archive_name}" "${tmp_dir}/checksums.txt"

    # Extract
    info "Extracting..."
    tar -xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"

    # Install
    install_dir=$(detect_install_dir)
    info "Installing to ${install_dir}/${BINARY_NAME}..."

    if [ ! -w "$install_dir" ]; then
        warn "No write permission to ${install_dir}, trying with sudo..."
        sudo install -m 755 "${tmp_dir}/${BINARY_NAME}" "${install_dir}/${BINARY_NAME}"
    else
        install -m 755 "${tmp_dir}/${BINARY_NAME}" "${install_dir}/${BINARY_NAME}"
    fi

    # Verify installation
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        installed_version=$("$BINARY_NAME" --version 2>/dev/null || true)
        info "Successfully installed: ${installed_version}"
    else
        info "Installed to ${install_dir}/${BINARY_NAME}"

        # Check if install_dir is in PATH
        case ":$PATH:" in
            *":${install_dir}:"*) ;;
            *)
                warn "${install_dir} is not in your PATH"
                echo ""
                echo "Add it to your shell profile:"
                echo "  export PATH=\"${install_dir}:\$PATH\""
                echo ""
                ;;
        esac
    fi

    echo ""
    info "Planck ${version} installed successfully!"
    echo ""
    echo "  Get started:  planck"
    echo "  Update later: planck update"
    echo ""
}

main "$@"
