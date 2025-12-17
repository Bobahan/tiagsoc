package app

import (
	"context"
	"net/http"
	"social/internal/config"

	"github.com/k0kubun/pp"
)

type application struct {
	config  *config.Config
	server  *http.Server
	handler http.Handler
}

func New(config *config.Config, handler http.Handler) *application {
	return &application{
		config:  config,
		handler: handler,
	}
}

func (app *application) Run() error {
	app.server = &http.Server{
		Addr:         app.config.Address,
		Handler:      app.handler,
		ReadTimeout:  app.config.ReadTimeout,
		WriteTimeout: app.config.WriteTimeout,
		IdleTimeout:  app.config.IdleTimeout,
	}

	pp.Printf("Server has started at %s ", app.config.Address)
	if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (app *application) Stop(ctx context.Context) error {
	pp.Println("Graceful shutdown is starting...")

	if err := app.server.Shutdown(ctx); err != nil {
		pp.Printf("Failed to shutdown server: %v\n", err)
		return err
	}

	pp.Println("Server was successfully stopped")
	return nil
}
