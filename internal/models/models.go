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
	PublicKey    string    `json:"public_key" db:"public_key"`
	Endpoint     string    `json:"endpoint" db:"endpoint"`
	AllowedIPs   []string  `json:"allowed_ips" db:"allowed_ips"`
	LatestHandshake time.Time `json:"latest_handshake" db:"latest_handshake"`
	ReceiveBytes int64     `json:"receive_bytes" db:"receive_bytes"`
	TransmitBytes int64    `json:"transmit_bytes" db:"transmit_bytes"`
}

// Status represents the current WireGuard interface status.
type Status struct {
	Interface string `json:"interface"`
	PublicKey string `json:"public_key"`
	ListenPort int    `json:"listen_port"`
	Peers     []Peer `json:"peers"`
}
