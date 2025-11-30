package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/behummble/29-11-2025/internal/config"
	"github.com/behummble/29-11-2025/internal/handlers/http"
	"github.com/behummble/29-11-2025/internal/service"
	"github.com/behummble/29-11-2025/internal/storage"
	svc "github.com/kardianos/service"
)

type App struct {
	Server *http.Server
	Service *service.LinkService
	log *slog.Logger
}

func NewApp(cfg config.Config, log *slog.Logger) *App {
	storage := storage.NewStorage(cfg.Storage, log)
	log.Info("Storage init")
	service := service.NewService(storage, log)
	server := http.NewServer(log, cfg.Server, service)
	return &App{
		log: log,
		Server: server,
		Service: service,
	}
}

func(app *App) Start(svc.Service) error {
	app.work()
	return nil
}

func(app *App) Stop(svc.Service) error {
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()
	return app.stopApp(shutdownContext)
}

func(app *App) Shutdown(ctx context.Context) error {
	return app.stopApp(ctx)
}

func(app *App) Run() {
	app.work()
}

func(app *App) work() {
	go app.Server.Start()
	app.log.Info("Server is Up")
	go app.Service.ValidateCache()
}

func(app *App) stopApp(ctx context.Context) error {
	err := app.Server.Shutdown(ctx)
	if err != nil {
		app.log.Error("Error while shutdown server", slog.String("error", err.Error()))
	} else {
		app.log.Info("Server is Down")
	}

	err = app.Service.Shutdown(ctx)
	if err != nil {
		app.log.Error("Error while shutdown cache validation", slog.String("error", err.Error()))
	} else {
		app.log.Info("Service is Down")
	}
	return err
}