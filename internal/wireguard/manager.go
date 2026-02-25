package wireguard

import (
	"context"

	"github.com/ChronoCoders/sentra/internal/models"
)

type Manager interface {
	GetStatus(ctx context.Context) (*models.Status, error)
	ListPeers(ctx context.Context) ([]models.Peer, error)
	Close() error
}
