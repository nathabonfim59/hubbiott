#!/bin/bash
# Pre-remove script for Hubbiott package
# This script runs before the package is removed

# Stop and disable the service
if command -v systemctl >/dev/null 2>&1; then
    echo "Stopping hubbiott service..."
    systemctl stop hubbiott.service 2>/dev/null || true

    echo "Disabling hubbiott service..."
    systemctl disable hubbiott.service 2>/dev/null || true
fi

exit 0
