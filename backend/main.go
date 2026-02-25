package main

import (
"net/http"
"os"
"time"

"github.com/prometheus/client_golang/prometheus/promhttp"
"github.com/rs/zerolog"
"github.com/rs/zerolog/log"
)

func main() {
zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

log.Info().Msg("Starting VPN UI backend")
log.Info().Str("version", "1.0.0").Msg("Application version")

http.HandleFunc("/api/status", handleStatus)
http.HandleFunc("/api/health", handleHealth)
http.HandleFunc("/api/events", handleEvents)
http.HandleFunc("/api/logs", logsHandler)
http.HandleFunc("/api/restart", restartHandler)
http.Handle("/metrics", promhttp.Handler())

addr := "127.0.0.1:8080"
log.Info().Str("address", addr).Msg("Server listening")

if err := http.ListenAndServe(addr, nil); err != nil {
log.Fatal().Err(err).Msg("Server failed to start")
}
}
