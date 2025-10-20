#!/bin/sh
# Patman installer script
# Usage: curl -sSfL https://raw.githubusercontent.com/lucagez/patman/main/install.sh | sh

set -e

# Detect OS and architecture
get_os() {
    case "$(uname -s)" in
        Darwin) echo "Darwin" ;;
        Linux) echo "Linux" ;;
        FreeBSD) echo "FreeBSD" ;;
        *) echo "Unsupported operating system: $(uname -s)" >&2; exit 1 ;;
    esac
}

get_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "x86_64" ;;
        aarch64|arm64) echo "arm64" ;;
        armv7l) echo "armv7" ;;
        armv6l) echo "armv6" ;;
        *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
    esac
}

get_latest_version() {
    curl -sSf "https://api.github.com/repos/lucagez/patman/releases/latest" |
        grep '"tag_name":' |
        sed -E 's/.*"([^"]+)".*/\1/'
}

main() {
    OS=$(get_os)
    ARCH=$(get_arch)
    VERSION=${VERSION:-$(get_latest_version)}

    if [ -z "$VERSION" ]; then
        echo "Error: Could not determine latest version" >&2
        exit 1
    fi

    echo "Installing patman ${VERSION} for ${OS}_${ARCH}..."

    # Construct download URL
    # macOS uses universal binary (all), others use specific architecture
    if [ "$OS" = "Darwin" ]; then
        BINARY_NAME="patman_${OS}_all"
    else
        BINARY_NAME="patman_${OS}_${ARCH}"
    fi

    if [ "$OS" = "Windows" ]; then
        ARCHIVE_NAME="${BINARY_NAME}.zip"
    else
        ARCHIVE_NAME="${BINARY_NAME}.tar.gz"
    fi

    DOWNLOAD_URL="https://github.com/lucagez/patman/releases/download/${VERSION}/${ARCHIVE_NAME}"

    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    echo "Downloading from ${DOWNLOAD_URL}..."
    if command -v curl > /dev/null 2>&1; then
        curl -sSfL "$DOWNLOAD_URL" -o "${TMP_DIR}/${ARCHIVE_NAME}"
    elif command -v wget > /dev/null 2>&1; then
        wget -q "$DOWNLOAD_URL" -O "${TMP_DIR}/${ARCHIVE_NAME}"
    else
        echo "Error: curl or wget is required" >&2
        exit 1
    fi

    # Extract archive
    cd "$TMP_DIR"
    if [ "${ARCHIVE_NAME##*.}" = "zip" ]; then
        unzip -q "$ARCHIVE_NAME"
    else
        tar -xzf "$ARCHIVE_NAME"
    fi

    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    elif [ -w "$HOME/.local/bin" ]; then
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
    else
        INSTALL_DIR="$HOME/bin"
        mkdir -p "$INSTALL_DIR"
    fi

    echo "Installing patman to ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv patman "$INSTALL_DIR/patman"
        chmod +x "$INSTALL_DIR/patman"
    else
        echo "Permission denied. Trying with sudo..."
        sudo mv patman "$INSTALL_DIR/patman"
        sudo chmod +x "$INSTALL_DIR/patman"
    fi

    echo "Successfully installed patman to ${INSTALL_DIR}/patman"

    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *)
            echo ""
            echo "WARNING: ${INSTALL_DIR} is not in your PATH"
            echo "Add it to your PATH by adding this line to your shell profile:"
            echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
            ;;
    esac

    echo ""
    echo "Run 'patman --help' to get started!"
}

main
