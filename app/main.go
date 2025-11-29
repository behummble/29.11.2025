package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"
	"syscall"

	"github.com/behummble/29-11-2025/internal/config"
	"github.com/behummble/29-11-2025/internal/handlers/http"
	"github.com/behummble/29-11-2025/internal/service"
	"github.com/behummble/29-11-2025/internal/storage"
	"github.com/behummble/29-11-2025/internal/app"
	svc "github.com/kardianos/service"
)

func main() {
	app := app.NewApp()
	s, err := registerService(app)
	if err != nil {
		// not service
		app.Run()
	} else {
		s.Run()
	}
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
	storage := storage.NewStorage(cfg.Storage, log)
	log.Info("Storage init")
	shutdownChan := make(chan struct{},1)
	service := service.NewService(storage, log, shutdownChan)
	server := http.NewServer(ctx, log, cfg.Server, service)
	go server.Start()
	log.Info("Server is Up")
	<- ctx.Done()
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()
	server.Shutdown(shutdownContext)
	log.Info("Server is Down")
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
	svcConfig := &svc.Config{
		Name:        "GoServiceExampleSimple",
		DisplayName: "Go Service Example",
		Description: "This is an example Go service.",
	}

	return  svc.New(app, svcConfig)
}