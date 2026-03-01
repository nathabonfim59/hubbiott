#!/bin/bash
# Post-install script for Hubbiott package
# This script runs after the package is installed

set -e

# Set ownership on directories (may fail if directories don't exist yet)
chown -R hubbiott:hubbiott /etc/hubbiott 2>/dev/null || true
chown -R hubbiott:hubbiott /var/lib/hubbiott 2>/dev/null || true

# Handle systemd
if command -v systemctl >/dev/null 2>&1; then
    echo "Reloading systemd daemon..."
    systemctl daemon-reload

    echo "Enabling hubbiott service..."
    systemctl enable hubbiott.service

    # Start service if config exists
    if [ -f /etc/hubbiott/config.yaml ]; then
        echo "Configuration found, starting hubbiott service..."
        systemctl start hubbiott.service || true
    fi
fi

echo ""
echo "=========================================="
echo "  Hubbiott installed successfully!"
echo "=========================================="
echo ""
echo "To complete setup:"
echo "  1. Copy the example config:"
echo "     cp /etc/hubbiott/config.example.yaml /etc/hubbiott/config.yaml"
echo ""
echo "  2. Edit the configuration file:"
echo "     nano /etc/hubbiott/config.yaml"
echo ""
echo "  3. Start the service:"
echo "     systemctl start hubbiott"
echo ""
echo "  4. Check status:"
echo "     systemctl status hubbiott"
echo ""

exit 0
