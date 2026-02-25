package main

import (
"github.com/prometheus/client_golang/prometheus"
"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
httpRequestsTotal = promauto.NewCounterVec(
prometheus.CounterOpts{
Name: "vpn_ui_http_requests_total",
Help: "Total number of HTTP requests",
},
[]string{"method", "path", "status"},
)

httpRequestDuration = promauto.NewHistogramVec(
prometheus.HistogramOpts{
Name:    "vpn_ui_http_request_duration_seconds",
Help:    "HTTP request duration in seconds",
Buckets: prometheus.DefBuckets,
},
[]string{"method", "path"},
)

wireguardPeersActive = promauto.NewGauge(
prometheus.GaugeOpts{
Name: "vpn_ui_wireguard_peers_active",
Help: "Number of active WireGuard peers",
},
)

wireguardBytesReceived = promauto.NewGauge(
prometheus.GaugeOpts{
Name: "vpn_ui_wireguard_bytes_received_total",
Help: "Total bytes received via WireGuard",
},
)

wireguardBytesTransmitted = promauto.NewGauge(
prometheus.GaugeOpts{
Name: "vpn_ui_wireguard_bytes_transmitted_total",
Help: "Total bytes transmitted via WireGuard",
},
)

hostCPUPercent = promauto.NewGauge(
prometheus.GaugeOpts{
Name: "vpn_ui_host_cpu_percent",
Help: "Host CPU usage percentage",
},
)

hostMemoryUsedMB = promauto.NewGauge(
prometheus.GaugeOpts{
Name: "vpn_ui_host_memory_used_mb",
Help: "Host memory used in MB",
},
)

hostDiskUsedGB = promauto.NewGauge(
prometheus.GaugeOpts{
Name: "vpn_ui_host_disk_used_gb",
Help: "Host disk used in GB",
},
)
)

func updateWireGuardMetrics(status Status) {
wireguardPeersActive.Set(float64(len(status.Peers)))

var totalRX, totalTX int64
for _, peer := range status.Peers {
totalRX += peer.RX
totalTX += peer.TX
}

wireguardBytesReceived.Set(float64(totalRX))
wireguardBytesTransmitted.Set(float64(totalTX))
}

func updateHostMetrics(health HealthResponse) {
hostCPUPercent.Set(health.Host.CPUPercent)
hostMemoryUsedMB.Set(float64(health.Host.MemUsedMB))
hostDiskUsedGB.Set(health.Host.DiskUsedGB)
}
