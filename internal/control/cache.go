package control

import (
	"context"
	"sync"

	"github.com/ChronoCoders/sentra/internal/models"
)

type StatusCache struct {
	mu      sync.RWMutex
	current *models.Status
	bus     *EventBus
}

func NewStatusCache(bus *EventBus) *StatusCache {
	c := &StatusCache{
		bus: bus,
	}
	go c.listen()
	return c
}

func (c *StatusCache) listen() {
	ch := c.bus.Subscribe()
	for event := range ch {
		c.mu.Lock()
		c.current = event.Status
		c.mu.Unlock()
	}
}

func (c *StatusCache) GetStatus(ctx context.Context) (*models.Status, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.current == nil {
		return nil, nil
	}
	return c.current, nil
}

func (c *StatusCache) ListPeers(ctx context.Context) ([]models.Peer, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.current == nil {
		return nil, nil
	}
	return c.current.Peers, nil
}
