#!/bin/bash
# Pre-install script for Hubbiott package
# This script runs before the package is installed

set -e

# Create hubbiott system user if it doesn't exist
if ! id -u hubbiott >/dev/null 2>&1; then
    echo "Creating hubbiott system user..."
    useradd --system --no-create-home --shell /bin/false hubbiott
fi

# Create configuration directory
mkdir -p /etc/hubbiott

# Create data directory
mkdir -p /var/lib/hubbiott

exit 0
