package main

import (
	"context"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ChronoCoders/sentra/internal/agent"
	"github.com/ChronoCoders/sentra/internal/api"
	"github.com/ChronoCoders/sentra/internal/config"
	"github.com/ChronoCoders/sentra/internal/control"
	"github.com/ChronoCoders/sentra/internal/store"
	sentratls "github.com/ChronoCoders/sentra/internal/tls"
	"github.com/ChronoCoders/sentra/internal/wireguard"
	"github.com/ChronoCoders/sentra/internal/ws"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	cfg := config.Load()

	// Init DB
	db, err := store.New(cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init database")
	}
	defer db.Close()

	// Init WG Manager
	wg, err := wireguard.NewWGManager(cfg.WGInterface)
	if err != nil {
		// Log error but continue for Control plane as it might be just a dashboard/management server
		// However, the embedded agent will fail to report WG status if this fails.
		// User instruction: "The system must fail clearly (log error and continue without peers) instead of generating fake peers."
		log.Error().Err(err).Msg("failed to init wireguard manager - local agent reporting will be limited")
	} else {
		defer wg.Close()
		// Verify if interface is accessible
		if _, err := wg.GetStatus(context.Background()); err != nil {
			log.Error().Err(err).Msg("failed to get status from wireguard interface - local agent reporting will be limited")
		}
	}

	// Init EventBus
	bus := control.NewEventBus()

	// Init WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// Init StatusCache (Client)
	client := control.NewStatusCache(bus, hub)

	// Init Agent
	if !cfg.DisableAgent {
		reporter := agent.NewEventBusReporter(bus)
		var ag *agent.Agent
		if wg != nil {
			ag = agent.New(wg, reporter, "local")
		} else {
			ag = agent.New(nil, reporter, "local")
		}
		go func() {
			if err := ag.Run(context.Background()); err != nil {
				log.Error().Err(err).Msg("agent run error")
			}
		}()
	} else {
		log.Info().Msg("internal agent disabled by configuration")
	}

	// Init API Server
	srv := api.NewServer(cfg, db, client, hub, bus)

	// Filter out noisy TLS handshake errors for internal agent reporting
	httpServer := &http.Server{
		Addr:     ":" + cfg.Port,
		Handler:  srv,
		ErrorLog: stdlog.New(&tlsErrorFilter{}, "", 0),
	}

	// Graceful shutdown
	go func() {
		if cfg.TLSAuto {
			if cfg.TLSCert == "" {
				cfg.TLSCert = "cert.pem"
			}
			if cfg.TLSKey == "" {
				cfg.TLSKey = "key.pem"
			}

			if _, err := os.Stat(cfg.TLSCert); os.IsNotExist(err) {
				log.Info().Msg("generating self-signed certificates")
				if err := sentratls.GenerateSelfSignedCert(cfg.TLSCert, cfg.TLSKey, cfg.TLSSANs); err != nil {
					log.Fatal().Err(err).Msg("failed to generate certificates")
				}
			}
		}

		if cfg.TLSCert != "" && cfg.TLSKey != "" {
			log.Info().Str("port", cfg.Port).Str("cert", cfg.TLSCert).Msg("starting control server (HTTPS)")
			if err := httpServer.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey); err != nil && err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("server error")
			}
		} else {
			log.Info().Str("port", cfg.Port).Msg("starting control server (HTTP)")
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("server error")
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server exited")
}

// tlsErrorFilter filters out noisy TLS handshake errors from http.Server logs
type tlsErrorFilter struct{}

func (f *tlsErrorFilter) Write(p []byte) (n int, err error) {
	msg := string(p)
	// Filter out "unknown certificate" and "remote error" which are common with self-signed certs
	// and browser/client probes or when user skips verification.
	if strings.Contains(msg, "TLS handshake error") && (strings.Contains(msg, "unknown certificate") || strings.Contains(msg, "remote error")) {
		return len(p), nil
	}
	// Forward other errors to stderr
	return os.Stderr.Write(p)
}
