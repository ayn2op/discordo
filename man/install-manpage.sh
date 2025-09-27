#!/bin/bash

# Discordo man page installer
# Usage: ./man/install-manpage.sh [--user]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MAN_SOURCE="$SCRIPT_DIR/discordo.1"
MAN_SECTION="1"

# Default installation directory (system-wide)
MAN_DIR="/usr/local/share/man/man$MAN_SECTION"

if [[ "$1" == "--user" ]]; then
    # User-specific installation
    if [[ -n "$HOME" ]]; then
        MAN_DIR="$HOME/.local/share/man/man$MAN_SECTION"
    else
        echo "Error: HOME environment variable not set"
        exit 1
    fi
fi

# Check if man page source exists
if [[ ! -f "$MAN_SOURCE" ]]; then
    echo "Error: Man page source not found at $MAN_SOURCE"
    exit 1
fi

# Create destination directory
echo "Creating directory: $MAN_DIR"
mkdir -p "$MAN_DIR"

# Install man page
MAN_DEST="$MAN_DIR/discordo.$MAN_SECTION"
echo "Installing man page to: $MAN_DEST"
cp "$MAN_SOURCE" "$MAN_DEST"

# Set appropriate permissions
chmod 644 "$MAN_DEST"

# Update man database
echo "Updating man database..."
if command -v mandb >/dev/null 2>&1; then
    if [[ "$1" == "--user" ]]; then
        mandb ~/.local/share/man
    else
        sudo mandb
    fi
elif command -v makewhatis >/dev/null 2>&1; then
    if [[ "$1" == "--user" ]]; then
        makewhatis ~/.local/share/man
    else
        sudo makewhatis
    fi
else
    echo "Warning: Neither 'mandb' nor 'makewhatis' found. Man database not updated."
    echo "You may need to run 'mandb' manually."
fi

echo "Man page installed successfully!"
echo "You can now view it with: man discordo"
