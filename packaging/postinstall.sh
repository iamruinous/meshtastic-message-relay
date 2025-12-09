#!/bin/sh
set -e

# Create system user if it doesn't exist
if ! getent passwd meshtastic-relay > /dev/null 2>&1; then
    useradd --system --no-create-home --shell /usr/sbin/nologin meshtastic-relay
fi

# Add user to dialout group for serial port access
if getent group dialout > /dev/null 2>&1; then
    usermod -a -G dialout meshtastic-relay 2>/dev/null || true
fi

# Set ownership of directories
chown -R meshtastic-relay:meshtastic-relay /var/log/meshtastic-relay 2>/dev/null || true

# Reload systemd
if command -v systemctl > /dev/null 2>&1; then
    systemctl daemon-reload
fi

echo "Meshtastic Relay installed successfully."
echo "To configure, copy /etc/meshtastic-relay/config.yaml.example to /etc/meshtastic-relay/config.yaml"
echo "Then start with: systemctl start meshtastic-relay"
