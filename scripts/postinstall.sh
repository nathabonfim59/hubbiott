#!/bin/bash
# Post-install script

# Set permissions
chown -R hubbiott:hubbiott /etc/hubbiott
chown -R hubbiott:hubbiott /var/lib/hubbiott

# Reload systemd
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload

    # Enable service if not already enabled
    systemctl enable hubbiott.service

    # Start service if config exists
    if [ -f /etc/hubbiott/config.yaml ]; then
        systemctl start hubbiott.service
    fi
fi

echo "Hubbiott installed successfully!"
echo ""
echo "To configure:"
echo "  1. Copy /etc/hubbiott/config.example.yaml to /etc/hubbiott/config.yaml"
echo "  2. Edit the configuration file with your settings"
echo "  3. Run: systemctl start hubbiott"
