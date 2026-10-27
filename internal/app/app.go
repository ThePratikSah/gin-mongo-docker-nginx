// Package app is the composition root: the ONLY place that knows about every
// concrete type and wires them together. Dependencies flow one way:
//
//	config -> database -> repository -> service -> handler -> router -> server
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ginmongo/internal/config"
	"ginmongo/internal/database"
	"ginmongo/internal/handler"
	"ginmongo/internal/repository"
	"ginmongo/internal/router"
	"ginmongo/internal/service"
)

type App struct {
	cfg    config.Config
	log    *slog.Logger
	db     *database.Mongo
	server *http.Server
}

func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	db, err := database.Connect(ctx, cfg.Mongo)
	if err != nil {
		return nil, err
	}

	engine, err := buildEngine(ctx, cfg, log, db)
	if err != nil {
		_ = db.Close(context.Background())
		return nil, err
	}

	srv := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	return &App{cfg: cfg, log: log, db: db, server: srv}, nil
}

// buildEngine wires every layer by hand. Adding a feature = add a repository,
// service and handler here, then pass the handler to router.New.
func buildEngine(ctx context.Context, cfg config.Config, log *slog.Logger, db *database.Mongo) (*gin.Engine, error) {
	// Repository layer
	userRepo := repository.NewUserRepository(db.DB())

	idxCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := userRepo.EnsureIndexes(idxCtx); err != nil {
		return nil, err
	}

	// Service layer
	userSvc := service.NewUserService(userRepo, log, time.Now)

	// Handler layer
	userHandler := handler.NewUserHandler(userSvc)
	healthHandler := handler.NewHealthHandler(db)

	// Transport
	engine, err := router.New(
		router.Options{
			Production:     cfg.Env == "production",
			TrustedProxies: cfg.HTTP.TrustedProxies,
		},
		log,
		healthHandler,
		userHandler,
	)
	if err != nil {
		return nil, fmt.Errorf("build router: %w", err)
	}
	return engine, nil
}

// Run serves HTTP until ctx is cancelled, then shuts down gracefully.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.log.Info("http server listening", "addr", a.server.Addr, "env", a.cfg.Env)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		_ = a.db.Close(context.Background())
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
	defer cancel()

	err := a.server.Shutdown(shutdownCtx)
	if cerr := a.db.Close(shutdownCtx); err == nil {
		err = cerr
	}
	a.log.Info("shutdown complete")
	return err
}
