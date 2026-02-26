package models

import "time"

// Organization represents a group of users and servers.
type Organization struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// User represents an authenticated user.
type User struct {
	ID        string    `json:"id" db:"id"`
	OrgID     string    `json:"org_id" db:"org_id"`
	Email     string    `json:"email" db:"email"`
	Name      string    `json:"name" db:"name"`
	Password  string    `json:"-" db:"password"` // New field
	Role      string    `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Server represents a VPN server (Control Plane).
type Server struct {
	ID        string    `json:"id" db:"id"`
	OrgID     string    `json:"org_id" db:"org_id"`
	Hostname  string    `json:"hostname" db:"hostname"`
	PublicKey string    `json:"public_key" db:"public_key"`
	Endpoint  string    `json:"endpoint" db:"endpoint"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Peer represents a WireGuard client.
type Peer struct {
	PublicKey       string    `json:"public_key" db:"public_key"`
	Endpoint        string    `json:"endpoint" db:"endpoint"`
	AllowedIPs      []string  `json:"allowed_ips" db:"allowed_ips"`
	LatestHandshake time.Time `json:"latest_handshake" db:"latest_handshake"`
	ReceiveBytes    int64     `json:"receive_bytes" db:"receive_bytes"`
	TransmitBytes   int64     `json:"transmit_bytes" db:"transmit_bytes"`
	KeepAlive       int       `json:"persistent_keepalive" db:"persistent_keepalive"` // Interval in seconds
}

// SystemInfo holds system metrics.
type SystemInfo struct {
	Hostname      string  `json:"hostname"`
	OS            string  `json:"os"`
	Arch          string  `json:"arch"`
	KernelVersion string  `json:"kernel_version"` // Added field
	Platform      string  `json:"platform"`       // Added field
	CPUCount      int     `json:"cpu_count"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryTotal   uint64  `json:"memory_total"`
	MemoryUsed    uint64  `json:"memory_used"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskTotal     uint64  `json:"disk_total"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskPercent   float64 `json:"disk_percent"`
	LoadAverage   float64 `json:"load_average"`
	Uptime        uint64  `json:"uptime"`
	NetBytesSent  uint64  `json:"net_bytes_sent"`
	NetBytesRecv  uint64  `json:"net_bytes_recv"`
}

// Status represents the current WireGuard interface status and system metrics.
type Status struct {
	Interface  string     `json:"interface"`
	PublicKey  string     `json:"public_key"`
	ListenPort int        `json:"listen_port"`
	Peers      []Peer     `json:"peers"`
	System     SystemInfo `json:"system"`
}
