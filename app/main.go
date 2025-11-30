package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/behummble/29-11-2025/internal/app"
	"github.com/behummble/29-11-2025/internal/config"
	svc "github.com/kardianos/service"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(), 
		os.Interrupt,
		syscall.SIGTERM,
        syscall.SIGINT,
        syscall.SIGQUIT,
	)
	defer stop()
	cfg := config.MustLoad()
	log := newLog(cfg.Log)

	app := app.NewApp(cfg, log)
	s, err := registerService(app)
	if err != nil {
		app.Run()
		<- ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
		defer cancel()
		app.Shutdown(shutdownContext)
	} else {
		s.Run()
	}
}

func newLog(config config.LogConfig) *slog.Logger {
	var output *os.File
	if config.Path != "" {
		file, err := os.OpenFile(config.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		output = file
	} else {
		output = os.Stdout
	}
	return slog.New(
		slog.NewJSONHandler(
			output,
			&slog.HandlerOptions{Level: slog.Level(config.Level)},
		),
	)
}

func registerService(app *app.App) (svc.Service, error) {
	if len(os.Args) <= 1 {
		return nil, errors.New("NotEnoughArguments")
	}
	svcConfig := &svc.Config{
		Name:        "links_verifier",
		DisplayName: "Links verifier service",
		Description: "Service verify sites condition",
	}

	serv, err := svc.New(app, svcConfig)
	if err != nil {
		return serv, err
	}
	return serv, svc.Control(serv, os.Args[1])
}