package wireguard

import (
	"context"
	"fmt"

	"github.com/ChronoCoders/sentra/internal/models"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type WGManager struct {
	client *wgctrl.Client
	iface  string
}

func NewWGManager(iface string) (*WGManager, error) {
	c, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("failed to open wgctrl: %w", err)
	}
	return &WGManager{client: c, iface: iface}, nil
}

func (m *WGManager) Close() error {
	return m.client.Close()
}

func (m *WGManager) GetStatus(ctx context.Context) (*models.Status, error) {
	d, err := m.client.Device(m.iface)
	if err != nil {
		return nil, fmt.Errorf("failed to get device %s: %w", m.iface, err)
	}

	peers := make([]models.Peer, len(d.Peers))
	for i, p := range d.Peers {
		peers[i] = mapPeer(p)
	}

	return &models.Status{
		Interface: d.Name,
		PublicKey: d.PublicKey.String(),
		ListenPort: d.ListenPort,
		Peers:     peers,
	}, nil
}

func (m *WGManager) ListPeers(ctx context.Context) ([]models.Peer, error) {
	d, err := m.client.Device(m.iface)
	if err != nil {
		return nil, fmt.Errorf("failed to get device %s: %w", m.iface, err)
	}

	peers := make([]models.Peer, len(d.Peers))
	for i, p := range d.Peers {
		peers[i] = mapPeer(p)
	}
	return peers, nil
}

func mapPeer(p wgtypes.Peer) models.Peer {
	allowedIPs := make([]string, len(p.AllowedIPs))
	for i, ip := range p.AllowedIPs {
		allowedIPs[i] = ip.String()
	}

	endpoint := ""
	if p.Endpoint != nil {
		endpoint = p.Endpoint.String()
	}

	return models.Peer{
		PublicKey:       p.PublicKey.String(),
		Endpoint:        endpoint,
		AllowedIPs:      allowedIPs,
		LatestHandshake: p.LastHandshakeTime,
		ReceiveBytes:    p.ReceiveBytes,
		TransmitBytes:   p.TransmitBytes,
		KeepAlive:       int(p.PersistentKeepaliveInterval.Seconds()),
	}
}
