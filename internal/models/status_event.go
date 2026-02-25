package models

import "time"

type StatusEvent struct {
	ServerID string
	Status   *Status
	Time     time.Time
}
