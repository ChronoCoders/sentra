package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ChronoCoders/sentra/internal/control"
	"github.com/ChronoCoders/sentra/internal/models"
	"github.com/rs/zerolog/log"
)

// Reporter defines how the agent reports status to the control plane.
type Reporter interface {
	Report(ctx context.Context, event models.StatusEvent) error
}

// EventBusReporter reports status to a local EventBus (for single-server mode).
type EventBusReporter struct {
	bus *control.EventBus
}

func NewEventBusReporter(bus *control.EventBus) *EventBusReporter {
	return &EventBusReporter{bus: bus}
}

func (r *EventBusReporter) Report(ctx context.Context, event models.StatusEvent) error {
	r.bus.Publish(event)
	return nil
}

// HTTPReporter reports status to a remote Control Plane via HTTP.
type HTTPReporter struct {
	serverURL string
	token     string
	client    *http.Client
}

func NewHTTPReporter(serverURL, token string, insecure bool) *HTTPReporter {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		log.Info().Msg("insecure mode enabled: skipping TLS verification")
	}

	return &HTTPReporter{
		serverURL: serverURL,
		token:     token,
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: tr,
		},
	}
}

func (r *HTTPReporter) Report(ctx context.Context, event models.StatusEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/report", r.serverURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}
