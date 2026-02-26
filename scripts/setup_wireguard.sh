#!/bin/bash
set -e

# Ensure clean slate
wg-quick down wg1 2>/dev/null || true
ip link delete wg1 2>/dev/null || true

# Generate keys if they don't exist
mkdir -p /etc/wireguard
if [ ! -f /etc/wireguard/server_private.key ]; then
    umask 077
    wg genkey | tee /etc/wireguard/server_private.key | wg pubkey > /etc/wireguard/server_public.key
fi
SERVER_PRIV=$(cat /etc/wireguard/server_private.key)
SERVER_PUB=$(cat /etc/wireguard/server_public.key)

# Generate client keys always fresh
umask 077
wg genkey | tee /etc/wireguard/client_private.key | wg pubkey > /etc/wireguard/client_public.key
CLIENT_PRIV=$(cat /etc/wireguard/client_private.key)
CLIENT_PUB=$(cat /etc/wireguard/client_public.key)

# Enable forwarding
sysctl -w net.ipv4.ip_forward=1
echo "net.ipv4.ip_forward=1" > /etc/sysctl.d/99-wireguard.conf

# Create wg1.conf
cat > /etc/wireguard/wg1.conf <<EOF
[Interface]
Address = 10.8.1.1/24
SaveConfig = true
PostUp = iptables -A FORWARD -i wg1 -j ACCEPT; iptables -t nat -A POSTROUTING -o enp1s0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg1 -j ACCEPT; iptables -t nat -D POSTROUTING -o enp1s0 -j MASQUERADE
ListenPort = 51822
PrivateKey = $SERVER_PRIV

[Peer]
PublicKey = $CLIENT_PUB
AllowedIPs = 10.8.1.2/32
EOF

# Bring up wg1
wg-quick up wg1

# Display Client Config
echo "=================================================="
echo "Use this configuration for your client (Phone/Laptop):"
echo "=================================================="
echo "[Interface]"
echo "PrivateKey = $CLIENT_PRIV"
echo "Address = 10.8.1.2/32"
echo "DNS = 8.8.8.8"
echo ""
echo "[Peer]"
echo "PublicKey = $SERVER_PUB"
echo "Endpoint = 66.135.2.77:51822"
echo "AllowedIPs = 0.0.0.0/0"
echo "PersistentKeepalive = 25"
echo "=================================================="
