#!/bin/sh
# Hubbiott Install Script
# curl -fsSL https://raw.githubusercontent.com/nathabonfim59/hubbiott/main/embed/install.sh | sh
#
# This script downloads and installs hubbiott from GitHub releases.
# It detects the OS and architecture, downloads the appropriate binary,
# verifies the checksum, and installs it to the appropriate location.

set -e

# Configuration
GITHUB_OWNER="nathabonfim59"
GITHUB_REPO="hubbiott"
BINARY_NAME="hubbiott"
INSTALL_VERSION="${INSTALL_VERSION:-latest}"

# Colors (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Logging functions
log_info() {
    printf "${BLUE}[INFO]${NC} %s\n" "$1"
}

log_success() {
    printf "${GREEN}[OK]${NC} %s\n" "$1"
}

log_warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

log_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1" >&2
}

# Cleanup temporary files on exit
cleanup() {
    if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
        rm -rf "$TMP_DIR"
    fi
}
trap cleanup EXIT

# Detect operating system
detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Linux*)
            OS="Linux"
            ;;
        Darwin*)
            OS="Darwin"
            ;;
        FreeBSD*)
            OS="FreeBSD"
            ;;
        *)
            log_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac
    log_info "Detected OS: $OS"
}

# Detect architecture
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)
            ARCH="x86_64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l|armhf)
            ARCH="armv7"
            ;;
        i386|i686)
            ARCH="i386"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    log_info "Detected architecture: $ARCH"
}

# Get the latest release version from GitHub
get_latest_version() {
    log_info "Fetching latest release version..."

    RELEASE_URL="https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest"

    # Use curl or wget
    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -fsSL "$RELEASE_URL" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        VERSION=$(wget -qO- "$RELEASE_URL" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')
    else
        log_error "Either curl or wget is required"
        exit 1
    fi

    if [ -z "$VERSION" ]; then
        log_error "Failed to get latest version"
        exit 1
    fi

    log_info "Latest version: $VERSION"
}

# Construct download URLs
construct_urls() {
    if [ "$INSTALL_VERSION" = "latest" ]; then
        get_latest_version
    else
        VERSION="$INSTALL_VERSION"
        log_info "Installing version: $VERSION"
    fi

    # Asset naming: {OS}_{arch}.tar.gz (e.g., Linux_x86_64.tar.gz)
    ASSET_NAME="${OS}_${ARCH}"
    DOWNLOAD_URL="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/v${VERSION}/${ASSET_NAME}.tar.gz"
    CHECKSUMS_URL="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/v${VERSION}/checksums.txt"
}

# Download file with curl or wget
download_file() {
    url="$1"
    output="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$url" -O "$output"
    else
        log_error "Either curl or wget is required"
        exit 1
    fi
}

# Download the archive and checksums
download() {
    TMP_DIR="$(mktemp -d)"
    ARCHIVE_PATH="${TMP_DIR}/${ASSET_NAME}.tar.gz"
    CHECKSUMS_PATH="${TMP_DIR}/checksums.txt"

    log_info "Downloading ${BINARY_NAME} v${VERSION} for ${OS}/${ARCH}..."

    download_file "$DOWNLOAD_URL" "$ARCHIVE_PATH"

    if [ ! -f "$ARCHIVE_PATH" ] || [ ! -s "$ARCHIVE_PATH" ]; then
        log_error "Failed to download archive"
        exit 1
    fi

    log_success "Download complete"

    # Try to download checksums (optional verification)
    if download_file "$CHECKSUMS_URL" "$CHECKSUMS_PATH" 2>/dev/null; then
        VERIFY_CHECKSUM=1
        log_info "Checksums downloaded"
    else
        VERIFY_CHECKSUM=0
        log_warn "Checksums not available, skipping verification"
    fi
}

# Verify SHA256 checksum
verify_checksum() {
    if [ "$VERIFY_CHECKSUM" != "1" ]; then
        return 0
    fi

    log_info "Verifying checksum..."

    # Get expected checksum from checksums file
    EXPECTED_CHECKSUM=$(grep "${ASSET_NAME}.tar.gz" "$CHECKSUMS_PATH" | awk '{print $1}')

    if [ -z "$EXPECTED_CHECKSUM" ]; then
        log_warn "Checksum not found in checksums file, skipping verification"
        return 0
    fi

    # Calculate actual checksum
    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL_CHECKSUM=$(sha256sum "$ARCHIVE_PATH" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL_CHECKSUM=$(shasum -a 256 "$ARCHIVE_PATH" | awk '{print $1}')
    else
        log_warn "No SHA256 tool available, skipping verification"
        return 0
    fi

    if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
        log_error "Checksum mismatch!"
        log_error "Expected: $EXPECTED_CHECKSUM"
        log_error "Actual:   $ACTUAL_CHECKSUM"
        exit 1
    fi

    log_success "Checksum verified"
}

# Extract the archive
extract() {
    log_info "Extracting archive..."

    EXTRACT_DIR="${TMP_DIR}/extract"
    mkdir -p "$EXTRACT_DIR"

    tar -xzf "$ARCHIVE_PATH" -C "$EXTRACT_DIR"

    BINARY_PATH="${EXTRACT_DIR}/${BINARY_NAME}"

    if [ ! -f "$BINARY_PATH" ]; then
        # Try to find the binary in subdirectories
        BINARY_PATH=$(find "$EXTRACT_DIR" -name "$BINARY_NAME" -type f | head -n 1)
    fi

    if [ ! -f "$BINARY_PATH" ]; then
        log_error "Binary not found in archive"
        exit 1
    fi

    log_success "Extraction complete"
}

# Determine install directory
get_install_dir() {
    # Check if running as root or with sudo
    if [ "$(id -u)" = "0" ]; then
        # System-wide installation
        if [ -d "/usr/local/bin" ]; then
            INSTALL_DIR="/usr/local/bin"
        elif [ -d "/usr/bin" ]; then
            INSTALL_DIR="/usr/bin"
        else
            INSTALL_DIR="/usr/local/bin"
        fi
    else
        # User installation
        if [ -d "$HOME/.local/bin" ]; then
            INSTALL_DIR="$HOME/.local/bin"
        elif [ -d "$HOME/bin" ]; then
            INSTALL_DIR="$HOME/bin"
        else
            # Create ~/.local/bin if it doesn't exist
            INSTALL_DIR="$HOME/.local/bin"
            mkdir -p "$INSTALL_DIR"
        fi
    fi

    log_info "Install directory: $INSTALL_DIR"
}

# Install the binary
install_binary() {
    get_install_dir

    INSTALL_PATH="${INSTALL_DIR}/${BINARY_NAME}"

    # Check if binary already exists
    if [ -f "$INSTALL_PATH" ]; then
        log_info "Removing existing installation..."
        rm -f "$INSTALL_PATH"
    fi

    # Copy binary
    cp "$BINARY_PATH" "$INSTALL_PATH"

    # Make executable
    chmod 755 "$INSTALL_PATH"

    log_success "Installed ${BINARY_NAME} to ${INSTALL_PATH}"
}

# Print success message and usage instructions
print_success() {
    printf "\n"
    printf "${GREEN}========================================${NC}\n"
    printf "${GREEN}  Hubbiott v${VERSION} installed successfully!${NC}\n"
    printf "${GREEN}========================================${NC}\n"
    printf "\n"

    # Check if install dir is in PATH
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        printf "${YELLOW}NOTE: ${INSTALL_DIR} is not in your PATH.${NC}\n"
        printf "\n"
        printf "Add it to your PATH by adding this line to your shell profile:\n"
        printf "\n"
        printf "    export PATH=\"\${PATH}:${INSTALL_DIR}\"\n"
        printf "\n"
        printf "Then reload your shell:\n"
        printf "\n"
        printf "    source ~/.bashrc  # or ~/.zshrc, ~/.profile, etc.\n"
        printf "\n"
    fi

    printf "Quick start:\n"
    printf "\n"
    printf "    ${BINARY_NAME} --help        Show available commands\n"
    printf "    ${BINARY_NAME} version       Show version information\n"
    printf "    ${BINARY_NAME} selfupdate    Update to the latest version\n"
    printf "\n"
    printf "Documentation: https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}\n"
    printf "\n"
}

# Main installation flow
main() {
    printf "\n"
    printf "${BLUE}Hubbiott Installer${NC}\n"
    printf "${BLUE}===================${NC}\n"
    printf "\n"

    detect_os
    detect_arch
    construct_urls
    download
    verify_checksum
    extract
    install_binary
    print_success
}

# Run main
main "$@"
