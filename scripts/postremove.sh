#!/bin/bash
# Post-remove script for Hubbiott package
# This script runs after the package is removed

# Reload systemd to remove the unit file reference
if command -v systemctl >/dev/null 2>&1; then
    echo "Reloading systemd daemon..."
    systemctl daemon-reload
fi

# Optional: Remove user and directories on purge
# Uncomment the following block if you want to clean up everything on purge
# if [ "$1" = "purge" ]; then
#     echo "Purging hubbiott data..."
#     userdel hubbiott 2>/dev/null || true
#     rm -rf /etc/hubbiott 2>/dev/null || true
#     rm -rf /var/lib/hubbiott 2>/dev/null || true
# fi

echo "Hubbiott has been removed."

exit 0
