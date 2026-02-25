package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ChronoCoders/sentra/internal/agent"
	"github.com/ChronoCoders/sentra/internal/api"
	"github.com/ChronoCoders/sentra/internal/config"
	"github.com/ChronoCoders/sentra/internal/control"
	"github.com/ChronoCoders/sentra/internal/store"
	"github.com/ChronoCoders/sentra/internal/wireguard"
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
		log.Fatal().Err(err).Msg("failed to init wireguard manager")
	}
	defer wg.Close()

	// Init EventBus
	bus := control.NewEventBus()

	// Init StatusCache (Client)
	client := control.NewStatusCache(bus)

	// Init Agent
	ag := agent.New(wg, bus, "local")
	go func() {
		if err := ag.Run(context.Background()); err != nil {
			log.Error().Err(err).Msg("agent run error")
		}
	}()

	// Init API Server
	srv := api.NewServer(cfg, db, client)

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: srv,
	}

	// Graceful shutdown
	go func() {
		log.Info().Str("port", cfg.Port).Msg("starting control server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
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
