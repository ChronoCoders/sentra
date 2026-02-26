package control

import (
	"context"
	"sync"

	"time"

	"github.com/ChronoCoders/sentra/internal/models"
)

type StatusBroadcaster interface {
	Broadcast(event models.StatusEvent)
}

type StatusCache struct {
	mu          sync.RWMutex
	statuses    map[string]*models.Status
	bus         *EventBus
	broadcaster StatusBroadcaster
}

func NewStatusCache(bus *EventBus, broadcaster StatusBroadcaster) *StatusCache {
	c := &StatusCache{
		bus:         bus,
		broadcaster: broadcaster,
		statuses:    make(map[string]*models.Status),
	}
	go c.listen()
	return c
}

func (c *StatusCache) listen() {
	ch := c.bus.Subscribe()
	for event := range ch {
		c.mu.Lock()
		c.statuses[event.ServerID] = event.Status
		c.mu.Unlock()
		if c.broadcaster != nil {
			c.broadcaster.Broadcast(event)
		}
	}
}

func (c *StatusCache) GetStatus(ctx context.Context, serverID string) (*models.Status, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.statuses[serverID], nil
}

func (c *StatusCache) GetAllStatuses() []models.StatusEvent {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var events []models.StatusEvent
	for id, status := range c.statuses {
		events = append(events, models.StatusEvent{
			ServerID: id,
			Status:   status,
			Time:     time.Now(),
		})
	}
	return events
}

func (c *StatusCache) ListPeers(ctx context.Context, serverID string) ([]models.Peer, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if status, ok := c.statuses[serverID]; ok {
		return status.Peers, nil
	}
	return nil, nil
}
