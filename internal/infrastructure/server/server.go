package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Run starts an HTTP server and shuts it down gracefully when ctx is canceled.
func Run(ctx context.Context, addr string, handler http.Handler, logger *slog.Logger) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil && logger != nil {
			logger.Error("server shutdown failed", "err", err)
		}
	}()

	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
