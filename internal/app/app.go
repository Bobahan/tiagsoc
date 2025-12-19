package app

import (
	"context"
	"net/http"
	"social/internal/config"

	"go.uber.org/zap"
)

type application struct {
	config  *config.Config
	server  *http.Server
	handler http.Handler
	logger  *zap.SugaredLogger
}

func New(config *config.Config, logger *zap.SugaredLogger, handler http.Handler) *application {
	return &application{
		config:  config,
		logger:  logger,
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

	app.logger.Infow("Server has been started", "addr", app.config.Address, "env:", app.config.Env)
	if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (app *application) Stop(ctx context.Context) error {
	app.logger.Info("Graceful shutdown is starting...")

	if err := app.server.Shutdown(ctx); err != nil {
		app.logger.Infof("Failed to shutdown server: %v\n", err)
		return err
	}

	app.logger.Info("Server was successfully stopped")
	return nil
}
