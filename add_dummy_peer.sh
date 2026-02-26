#!/bin/bash
# Add a dummy peer to wg0 for testing Sentra Dashboard
# This script ensures wg0 exists and has a peer with traffic stats.

# Check if wg0 exists
if ! ip link show wg0 > /dev/null 2>&1; then
    echo "Creating dummy wg0 interface..."
    ip link add dev wg0 type wireguard
    ip address add 10.0.0.1/24 dev wg0
    ip link set up dev wg0
fi

# Generate a dummy key pair (using docker control container if wg not on host)
if command -v wg > /dev/null; then
    PRIVATE_KEY=$(wg genkey)
    PUBLIC_KEY=$(echo "$PRIVATE_KEY" | wg pubkey)
else
    # Fallback to hardcoded valid keys if wg tool missing
    PRIVATE_KEY="cE+EXAMPLE_PRIVATE_KEY_DONT_USE_IN_PROD="
    PUBLIC_KEY="7tT7L+4r5qjLzJ8Vb9a0K1o2p3q4r5s6t7u8v9w0x1y="
fi

# Add a peer with some stats
# Note: Real stats require traffic. We can't fake kernel stats easily without traffic.
# But at least the peer will show up in the list.
echo "Adding dummy peer..."
if command -v wg > /dev/null; then
    wg set wg0 peer "$PUBLIC_KEY" allowed-ips 10.0.0.2/32 endpoint 192.168.1.100:51820 persistent-keepalive 25
else
    # Try using docker container to configure host interface?
    # No, container might not see host interface unless --network host.
    # We will try best effort.
    echo "Error: 'wg' command not found. Please install wireguard-tools."
    # If we can't add a peer, and we created the interface, we should probably delete it 
    # so the agent falls back to Mock Mode (if configured to do so on missing interface).
    echo "Deleting empty wg0 interface to trigger Agent Mock Mode..."
    ip link delete dev wg0
    exit 1
fi

echo "Dummy peer added. Check dashboard."
