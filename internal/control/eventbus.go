package control

import (
	"github.com/ChronoCoders/sentra/internal/models"
)

type EventBus struct {
	ch chan models.StatusEvent
}

func NewEventBus() *EventBus {
	return &EventBus{
		ch: make(chan models.StatusEvent, 100),
	}
}

func (b *EventBus) Publish(event models.StatusEvent) {
	select {
	case b.ch <- event:
	default:
		// Drop event if buffer full
	}
}

func (b *EventBus) Subscribe() <-chan models.StatusEvent {
	return b.ch
}
