package control

import (
	"context"

	"github.com/ChronoCoders/sentra/internal/models"
	"github.com/ChronoCoders/sentra/internal/wireguard"
)

type LocalClient struct {
	manager wireguard.Manager
}

func NewLocalClient(m wireguard.Manager) *LocalClient {
	return &LocalClient{manager: m}
}

func (c *LocalClient) GetStatus(ctx context.Context) (*models.Status, error) {
	return c.manager.GetStatus(ctx)
}

func (c *LocalClient) ListPeers(ctx context.Context) ([]models.Peer, error) {
	return c.manager.ListPeers(ctx)
}
