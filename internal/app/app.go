package app

import (
	"log/slog"
	"github.com/kardianos/service"
)

type App struct {
	Server Server
	Service Service
	log *slog.Logger
}

type Server interface {
	Start()
	Shutdown()
}

type Service interface {
	ValidateCache()
}

func NewApp() *App {
	return nil
}

func(app *App) Start(service.Service) error {
	app.Server.Start()
	app.Service.ValidateCache()
	return nil
}

func(app *App) Stop(service.Service) error {
	app.Server.Shutdown()
	return nil
}

func(app *App) Run() {

}