#!/bin/sh
# install.sh — installs the latest gitredact binary for your platform
set -e

REPO="haukened/gitredact"
INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="gitredact"

# ── helpers ──────────────────────────────────────────────────────────────────

die() { printf 'error: %s\n' "$1" >&2; exit 1; }
info() { printf '==> %s\n' "$1"; }

# ── detect OS ────────────────────────────────────────────────────────────────

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
    darwin|linux) ;;
    *) die "unsupported OS: $OS (supported: darwin, linux)" ;;
esac

# ── detect arch ──────────────────────────────────────────────────────────────

ARCH_RAW="$(uname -m)"
case "$ARCH_RAW" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) die "unsupported architecture: $ARCH_RAW (supported: x86_64, aarch64, arm64)" ;;
esac

info "Detected: $OS/$ARCH"

# ── fetch latest release tag ─────────────────────────────────────────────────

info "Fetching latest release tag..."
API_URL="https://api.github.com/repos/${REPO}/releases/latest"

if command -v curl >/dev/null 2>&1; then
    API_RESPONSE="$(curl -fsSL "$API_URL")"
elif command -v wget >/dev/null 2>&1; then
    API_RESPONSE="$(wget -qO- "$API_URL")"
else
    die "curl or wget is required but neither was found"
fi

TAG="$(printf '%s' "$API_RESPONSE" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"
[ -n "$TAG" ] || die "failed to parse release tag from GitHub API"

info "Latest release: $TAG"

# ── download binary ───────────────────────────────────────────────────────────

ASSET="gitredact-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"
DEST="${INSTALL_DIR}/${BINARY_NAME}"

info "Downloading $ASSET..."
mkdir -p "$INSTALL_DIR"

if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$DOWNLOAD_URL" -o "$DEST"
else
    wget -qO "$DEST" "$DOWNLOAD_URL"
fi

chmod +x "$DEST"
info "Installed: $DEST"

# ── PATH injection ────────────────────────────────────────────────────────────

PATH_EXPORT='export PATH="$HOME/.local/bin:$PATH"'
FISH_PATH='fish_add_path "$HOME/.local/bin"'

# Appends LINE to FILE if ~/.local/bin is not already mentioned in it.
# Returns 0 if the file was updated, 1 if it was already configured (no-op).
add_to_config() {
    _file="$1"
    _line="$2"
    if [ -f "$_file" ] && grep -q '\.local/bin' "$_file" 2>/dev/null; then
        return 1
    fi
    mkdir -p "$(dirname "$_file")"
    printf '\n# Added by gitredact installer\n%s\n' "$_line" >> "$_file"
    return 0
}

SHELL_CONF=""
SHELL_NAME="$(basename "${SHELL:-sh}")"

case "$SHELL_NAME" in
    bash)
        if [ -f "$HOME/.bashrc" ]; then
            TARGET="$HOME/.bashrc"
        elif [ -f "$HOME/.bash_profile" ]; then
            TARGET="$HOME/.bash_profile"
        else
            TARGET="$HOME/.bashrc"
        fi
        if add_to_config "$TARGET" "$PATH_EXPORT"; then
            SHELL_CONF="$TARGET"
        fi
        ;;
    zsh)
        TARGET="$HOME/.zshrc"
        if add_to_config "$TARGET" "$PATH_EXPORT"; then
            SHELL_CONF="$TARGET"
        fi
        ;;
    fish)
        TARGET="$HOME/.config/fish/config.fish"
        if add_to_config "$TARGET" "$FISH_PATH"; then
            SHELL_CONF="$TARGET"
        fi
        ;;
    *)
        TARGET="$HOME/.profile"
        if add_to_config "$TARGET" "$PATH_EXPORT"; then
            SHELL_CONF="$TARGET"
        fi
        ;;
esac

# ── summary ───────────────────────────────────────────────────────────────────

printf '\n'
printf 'gitredact %s installed successfully!\n' "$TAG"
printf '\n'
printf '  Binary: %s\n' "$DEST"

if [ -n "$SHELL_CONF" ]; then
    printf '  Shell config updated: %s\n' "$SHELL_CONF"
    printf '\n'
    printf 'Reload your shell to use gitredact:\n'
    case "$SHELL_NAME" in
        fish) printf '  exec %s\n' "$SHELL" ;;
        *)    printf '  source %s\n' "$SHELL_CONF" ;;
    esac
else
    printf '  PATH: ~/.local/bin is already configured\n'
fi
printf '\n'
printf '  Run: gitredact --version\n'
printf '\n'
