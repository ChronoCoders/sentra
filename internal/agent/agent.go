package agent

import (
	"context"
	"time"

	"github.com/ChronoCoders/sentra/internal/control"
	"github.com/ChronoCoders/sentra/internal/models"
	"github.com/ChronoCoders/sentra/internal/wireguard"
	"github.com/rs/zerolog/log"
)

type Agent struct {
	wg       *wireguard.WGManager
	bus      *control.EventBus
	serverID string
}

func New(wg *wireguard.WGManager, bus *control.EventBus, serverID string) *Agent {
	return &Agent{wg: wg, bus: bus, serverID: serverID}
}

func (a *Agent) Run(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			status, err := a.wg.GetStatus(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to get status")
				continue
			}

			event := models.StatusEvent{
				ServerID: a.serverID,
				Status:   status,
				Time:     time.Now(),
			}
			a.bus.Publish(event)

			log.Info().Int("peer_count", len(status.Peers)).Msg("agent status report")
		}
	}
}
