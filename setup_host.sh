#!/bin/bash
# Sentra Host Network Optimization Script
# Run this on the host machine (not inside the container) to optimize network buffers for WireGuard.

echo "Applying network optimizations..."

# Optimize UDP buffers for WireGuard (prevents packet drops during bursts)
sysctl -w net.core.rmem_max=4194304
sysctl -w net.core.wmem_max=4194304
sysctl -w net.core.rmem_default=262144
sysctl -w net.core.wmem_default=262144

# Enable IP forwarding (required for VPN routing)
sysctl -w net.ipv4.ip_forward=1

# Enable BBR Congestion Control (improves TCP throughput/latency)
# Load module first
modprobe tcp_bbr
sysctl -w net.ipv4.tcp_congestion_control=bbr

echo "Optimizations applied temporarily. To make them persistent, add them to /etc/sysctl.conf"
