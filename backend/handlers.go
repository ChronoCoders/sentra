package main

import (
"encoding/json"
"net/http"
"time"

"github.com/rs/zerolog/log"
)

func handleStatus(w http.ResponseWriter, r *http.Request) {
start := time.Now()

status, err := getWGStatus()
if err != nil {
log.Error().Err(err).Msg("Failed to get WireGuard status")
httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "500").Inc()
http.Error(w, "Failed to get status", http.StatusInternalServerError)
return
}

updateWireGuardMetrics(status)

w.Header().Set("Content-Type", "application/json")
if err := json.NewEncoder(w).Encode(status); err != nil {
log.Error().Err(err).Msg("Failed to encode status response")
httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "500").Inc()
} else {
httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "200").Inc()
}

httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
start := time.Now()

health, err := getHealth()
if err != nil {
log.Error().Err(err).Msg("Failed to get health metrics")
httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "500").Inc()
http.Error(w, "Failed to get health", http.StatusInternalServerError)
return
}

updateHostMetrics(health)

w.Header().Set("Content-Type", "application/json")
if err := json.NewEncoder(w).Encode(health); err != nil {
log.Error().Err(err).Msg("Failed to encode health response")
httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "500").Inc()
} else {
httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "200").Inc()
}

httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
start := time.Now()

window := r.URL.Query().Get("window")
if window == "" {
window = "50"
}

events, err := getEvents(window)
if err != nil {
log.Error().Err(err).Str("window", window).Msg("Failed to get events")
httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "500").Inc()
http.Error(w, "Failed to get events", http.StatusInternalServerError)
return
}

w.Header().Set("Content-Type", "application/json")
if err := json.NewEncoder(w).Encode(events); err != nil {
log.Error().Err(err).Msg("Failed to encode events response")
httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "500").Inc()
} else {
httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "200").Inc()
}

httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
}
