package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ChronoCoders/sentra/internal/agent"
	"github.com/ChronoCoders/sentra/internal/config"
	"github.com/ChronoCoders/sentra/internal/control"
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
		log.Fatal().Err(err).Msg("failed to init wireguard manager")
	}
	defer wg.Close()

	// Init EventBus
	bus := control.NewEventBus()

	// Init Agent
	agt := agent.New(wg, bus, "standalone-agent")

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
