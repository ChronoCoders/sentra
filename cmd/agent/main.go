package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ChronoCoders/sentra/internal/agent"
	"github.com/ChronoCoders/sentra/internal/config"
	"github.com/ChronoCoders/sentra/internal/wireguard"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	cfg := config.Load()

	// Init WG Manager
	wg, err := wireguard.NewWGManager(cfg.WGInterface)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init wireguard manager. ensure interface exists and process has permissions")
	}
	defer wg.Close()

	// Verify if interface is accessible
	if _, err := wg.GetStatus(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to get status from wireguard interface. ensure interface is up")
	}

	// Init Reporter
	reporter := agent.NewHTTPReporter(cfg.ControlURL, cfg.AuthToken)

	// Init Agent
	agt := agent.New(wg, reporter, cfg.ServerID)

	// Run Agent
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := agt.Run(ctx); err != nil {
			log.Error().Err(err).Msg("agent error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down agent...")
	cancel()
	time.Sleep(1 * time.Second)
	log.Info().Msg("agent exited")
}
