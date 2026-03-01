#!/bin/bash
# Post-remove script

# Clean up systemd
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
fi

# Remove user on purge (optional - keep for now)
# if [ "$1" = "purge" ]; then
#     userdel hubbiott 2>/dev/null || true
# fi

exit 0
