#!/bin/bash
# Pre-install script

# Create hubbiott user if it doesn't exist
if ! id -u hubbiott >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /bin/false hubbiott
fi

# Create directories
mkdir -p /etc/hubbiott
mkdir -p /var/lib/hubbiott

exit 0
