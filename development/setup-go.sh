#!/bin/bash

# Minimum required version of Go
MIN_VERSION="1.22.3"

# Function to parse and compare Go versions
version_gt() { test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1"; }

# Check if Go is installed and get the version
if command -v go > /dev/null; then
    # Go is installed, check version
    CURRENT_VERSION=$(go version | awk '{print $3}' | cut -d 'o' -f2)
    echo "Go is installed. Current version is $CURRENT_VERSION"
    if version_gt $MIN_VERSION $CURRENT_VERSION; then
        echo "Installed Go version is less than the minimum required $MIN_VERSION"
        NEED_INSTALL=true
    else
        echo "Go meets the minimum version requirement."
        NEED_INSTALL=false
    fi
else
    echo "Go is not installed."
    NEED_INSTALL=true
fi

# Install or update Go if necessary
if [ "$NEED_INSTALL" = true ]; then
    echo "Installing or updating Go..."
    # Replace the URL with the appropriate version for your platform
    wget https://go.dev/dl/go${MIN_VERSION}.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go${MIN_VERSION}.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo "Go installed successfully."
fi
