package control

import (
	"context"

	"github.com/ChronoCoders/sentra/internal/models"
)

type AgentClient interface {
	GetStatus(ctx context.Context, serverID string) (*models.Status, error)
	ListPeers(ctx context.Context, serverID string) ([]models.Peer, error)
	GetAllStatuses() []models.StatusEvent
}
