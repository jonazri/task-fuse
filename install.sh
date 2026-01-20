#!/bin/bash
set -e

# task-fuse installer
# Detects OS and package manager, installs all dependencies including Go, and builds from source

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

GO_VERSION_REQUIRED="1.21"
GO_VERSION_INSTALL="1.23.5"

# Detect OS
detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Linux)
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                DISTRO="$ID"
                DISTRO_FAMILY="$ID_LIKE"
            elif [ -f /etc/debian_version ]; then
                DISTRO="debian"
            elif [ -f /etc/redhat-release ]; then
                DISTRO="rhel"
            elif [ -f /etc/arch-release ]; then
                DISTRO="arch"
            elif [ -f /etc/alpine-release ]; then
                DISTRO="alpine"
            else
                DISTRO="unknown"
            fi
            ;;
        Darwin)
            error "task-fuse requires Linux. macOS is not supported because the FUSE library (bazil.org/fuse) only works on Linux."
            ;;
        MINGW*|MSYS*|CYGWIN*)
            error "task-fuse requires Linux. Windows is not supported."
            ;;
        *)
            error "Unknown operating system: $OS. task-fuse requires Linux."
            ;;
    esac

    info "Detected OS: Linux ($DISTRO)"
}

# Detect package manager based on distro
detect_package_manager() {
    # Try to match distro or distro family
    case "$DISTRO" in
        ubuntu|debian|linuxmint|pop|elementary|zorin|kali|raspbian)
            PKG_MANAGER="apt"
            ;;
        fedora|rhel|centos|rocky|alma|oracle)
            if command -v dnf &> /dev/null; then
                PKG_MANAGER="dnf"
            else
                PKG_MANAGER="yum"
            fi
            ;;
        arch|manjaro|endeavouros|garuda)
            PKG_MANAGER="pacman"
            ;;
        opensuse*|suse|sles)
            PKG_MANAGER="zypper"
            ;;
        alpine)
            PKG_MANAGER="apk"
            ;;
        void)
            PKG_MANAGER="xbps"
            ;;
        gentoo)
            PKG_MANAGER="emerge"
            ;;
        *)
            # Fallback: detect by available commands
            if command -v apt-get &> /dev/null; then
                PKG_MANAGER="apt"
            elif command -v dnf &> /dev/null; then
                PKG_MANAGER="dnf"
            elif command -v yum &> /dev/null; then
                PKG_MANAGER="yum"
            elif command -v pacman &> /dev/null; then
                PKG_MANAGER="pacman"
            elif command -v zypper &> /dev/null; then
                PKG_MANAGER="zypper"
            elif command -v apk &> /dev/null; then
                PKG_MANAGER="apk"
            elif command -v xbps-install &> /dev/null; then
                PKG_MANAGER="xbps"
            elif command -v emerge &> /dev/null; then
                PKG_MANAGER="emerge"
            else
                PKG_MANAGER="unknown"
            fi
            ;;
    esac

    if [ "$PKG_MANAGER" = "unknown" ]; then
        error "Could not detect package manager. Please install dependencies manually:
  - Go 1.21+: https://go.dev/dl/
  - FUSE: fuse and libfuse-dev (or equivalent for your distro)"
    fi

    info "Detected package manager: $PKG_MANAGER"
}

# Install Go if not present or too old
install_go() {
    local need_install=false

    if ! command -v go &> /dev/null; then
        info "Go is not installed"
        need_install=true
    else
        # Extract version - handle different formats
        GO_VERSION=$(go version | sed -n 's/.*go\([0-9]*\.[0-9]*\).*/\1/p')
        if [ -z "$GO_VERSION" ]; then
            warn "Could not determine Go version"
            need_install=true
        elif [ "$(printf '%s\n' "$GO_VERSION_REQUIRED" "$GO_VERSION" | sort -V | head -n1)" != "$GO_VERSION_REQUIRED" ]; then
            info "Go $GO_VERSION is too old (need $GO_VERSION_REQUIRED+)"
            need_install=true
        else
            info "Found Go $GO_VERSION"
            return 0
        fi
    fi

    if [ "$need_install" = true ]; then
        info "Installing Go $GO_VERSION_INSTALL..."

        case "$PKG_MANAGER" in
            apt)
                # Ubuntu/Debian repos often have old Go, use official tarball
                install_go_from_tarball
                ;;
            dnf|yum)
                # Try package manager first, fall back to tarball
                if sudo $PKG_MANAGER install -y golang &> /dev/null; then
                    # Check if installed version is new enough
                    GO_VERSION=$(go version | sed -n 's/.*go\([0-9]*\.[0-9]*\).*/\1/p')
                    if [ "$(printf '%s\n' "$GO_VERSION_REQUIRED" "$GO_VERSION" | sort -V | head -n1)" != "$GO_VERSION_REQUIRED" ]; then
                        warn "Package manager Go is too old, installing from tarball..."
                        install_go_from_tarball
                    fi
                else
                    install_go_from_tarball
                fi
                ;;
            pacman)
                sudo pacman -S --noconfirm go
                ;;
            zypper)
                sudo zypper install -y go
                ;;
            apk)
                sudo apk add go
                ;;
            xbps)
                sudo xbps-install -y go
                ;;
            emerge)
                sudo emerge dev-lang/go
                ;;
            *)
                install_go_from_tarball
                ;;
        esac

        # Verify installation
        if ! command -v go &> /dev/null; then
            # Check if it's in the standard location
            if [ -x /usr/local/go/bin/go ]; then
                export PATH=$PATH:/usr/local/go/bin
                info "Added /usr/local/go/bin to PATH"
            else
                error "Go installation failed. Please install Go $GO_VERSION_REQUIRED+ manually: https://go.dev/dl/"
            fi
        fi

        GO_VERSION=$(go version | sed -n 's/.*go\([0-9]*\.[0-9]*\).*/\1/p')
        info "Installed Go $GO_VERSION"
    fi
}

# Install Go from official tarball
install_go_from_tarball() {
    local ARCH
    case "$(uname -m)" in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        armv7l|armv6l) ARCH="armv6l" ;;
        i686|i386) ARCH="386" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac

    local TARBALL="go${GO_VERSION_INSTALL}.linux-${ARCH}.tar.gz"
    local URL="https://go.dev/dl/${TARBALL}"

    info "Downloading Go from $URL..."

    # Download
    if command -v curl &> /dev/null; then
        curl -fsSL -o "/tmp/$TARBALL" "$URL"
    elif command -v wget &> /dev/null; then
        wget -q -O "/tmp/$TARBALL" "$URL"
    else
        # Install curl first
        case "$PKG_MANAGER" in
            apt) sudo apt-get update && sudo apt-get install -y curl ;;
            dnf|yum) sudo $PKG_MANAGER install -y curl ;;
            pacman) sudo pacman -S --noconfirm curl ;;
            zypper) sudo zypper install -y curl ;;
            apk) sudo apk add curl ;;
            xbps) sudo xbps-install -y curl ;;
            *) error "Please install curl or wget first" ;;
        esac
        curl -fsSL -o "/tmp/$TARBALL" "$URL"
    fi

    # Remove old Go and extract
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "/tmp/$TARBALL"
    rm "/tmp/$TARBALL"

    # Add to PATH for current session
    export PATH=$PATH:/usr/local/go/bin

    # Add to shell profile for future sessions
    if [ -n "$HOME" ]; then
        local PROFILE=""
        if [ -f "$HOME/.bashrc" ]; then
            PROFILE="$HOME/.bashrc"
        elif [ -f "$HOME/.bash_profile" ]; then
            PROFILE="$HOME/.bash_profile"
        elif [ -f "$HOME/.profile" ]; then
            PROFILE="$HOME/.profile"
        fi

        if [ -n "$PROFILE" ]; then
            if ! grep -q '/usr/local/go/bin' "$PROFILE" 2>/dev/null; then
                echo 'export PATH=$PATH:/usr/local/go/bin' >> "$PROFILE"
                info "Added Go to PATH in $PROFILE"
            fi
        fi
    fi
}

# Install FUSE dependencies
install_fuse() {
    info "Installing FUSE dependencies..."

    case "$PKG_MANAGER" in
        apt)
            sudo apt-get update
            sudo apt-get install -y fuse libfuse-dev build-essential
            ;;
        dnf)
            sudo dnf install -y fuse fuse-devel gcc make
            ;;
        yum)
            sudo yum install -y fuse fuse-devel gcc make
            ;;
        pacman)
            sudo pacman -S --noconfirm fuse2 base-devel
            ;;
        zypper)
            sudo zypper install -y fuse libfuse2 fuse-devel gcc make
            ;;
        apk)
            sudo apk add fuse fuse-dev build-base
            ;;
        xbps)
            sudo xbps-install -y fuse fuse-devel base-devel
            ;;
        emerge)
            sudo emerge sys-fs/fuse
            ;;
        *)
            warn "Could not install FUSE automatically. Please install manually:"
            echo "  - fuse (or fuse2)"
            echo "  - fuse development headers (libfuse-dev, fuse-devel, etc.)"
            return 1
            ;;
    esac

    info "FUSE dependencies installed"
}

# Build task-fuse
build() {
    info "Building task-fuse..."

    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    cd "$SCRIPT_DIR"

    go mod tidy
    go build -o task-fuse ./cmd/task-fuse

    info "Build successful: $SCRIPT_DIR/task-fuse"
}

# Install to /usr/local/bin
install_binary() {
    info "Installing to /usr/local/bin..."
    sudo cp task-fuse /usr/local/bin/
    info "Installed to /usr/local/bin/task-fuse"
}

# Main
main() {
    echo "=== task-fuse installer ==="
    echo

    detect_os
    detect_package_manager
    install_go
    install_fuse
    build
    install_binary

    echo
    info "Installation complete!"
    echo
    echo "Usage:"
    echo "  task-fuse mount tasks.md /mnt/tasks"
    echo "  task-fuse status"
    echo "  task-fuse unmount /mnt/tasks"
}

main "$@"
