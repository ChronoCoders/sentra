package control

import (
	"context"

	"github.com/ChronoCoders/sentra/internal/models"
)

type AgentClient interface {
	GetStatus(ctx context.Context) (*models.Status, error)
	ListPeers(ctx context.Context) ([]models.Peer, error)
}
