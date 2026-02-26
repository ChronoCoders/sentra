package models

import "time"

type StatusEvent struct {
	ServerID string    `json:"server_id"`
	OrgID    string    `json:"org_id"`
	Status   *Status   `json:"status"`
	Time     time.Time `json:"time"`
}
