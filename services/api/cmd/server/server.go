package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/tachigo/tachigo/internal/config"
	tracingprovider "github.com/tachigo/tachigo/internal/tracing"
	"gorm.io/gorm"
)

const (
	serverReadTimeout     = 15 * time.Second
	serverWriteTimeout    = 120 * time.Second
	serverIdleTimeout     = 60 * time.Second
	serverShutdownTimeout = 10 * time.Second
)

type gracefulHTTPServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}
}

func configureTracing(ctx context.Context, cfg *config.Config) (tracingprovider.ShutdownFunc, error) {
	if cfg == nil {
		return func(context.Context) error { return nil }, nil
	}
	return tracingprovider.ConfigureProvider(ctx, cfg.Tracing)
}

func closeServerResources(db *gorm.DB, tracingShutdown tracingprovider.ShutdownFunc) func() error {
	return func() error {
		if tracingShutdown == nil {
			tracingShutdown = func(context.Context) error { return nil }
		}
		return errors.Join(tracingShutdown(context.Background()), closeDatabase(db))
	}
}

func runHTTPServer(ctx context.Context, srv gracefulHTTPServer, closeDB func() error) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return finishHTTPServer(err, closeDB)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
		defer cancel()

		shutdownErr := srv.Shutdown(shutdownCtx)
		var serveErr error
		select {
		case serveErr = <-errCh:
		case <-shutdownCtx.Done():
			serveErr = shutdownCtx.Err()
		}

		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		return finishHTTPServer(errors.Join(shutdownErr, serveErr), closeDB)
	}
}

func finishHTTPServer(serverErr error, closeDB func() error) error {
	closeErr := runCloseHook(closeDB)
	if errors.Is(serverErr, http.ErrServerClosed) {
		serverErr = nil
	}
	return errors.Join(serverErr, closeErr)
}

func runCloseHook(closeDB func() error) error {
	if closeDB == nil {
		return nil
	}
	return closeDB()
}

func closeDatabase(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
