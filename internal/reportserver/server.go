package reportserver

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Config captures the settings for serving a DuckDB-backed report.
type Config struct {
	Addr          string
	DBPath        string
	AssetsBaseURL string
}

// Serve starts an HTTP server that hosts the report UI and data endpoints.
func Serve(ctx context.Context, cfg Config) error {
	if ctx == nil {
		return errors.New("reportserver: context is nil")
	}
	if cfg.Addr == "" {
		return errors.New("reportserver: addr is required")
	}
	handler, err := NewHandler(cfg)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: handler,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) || err == nil {
			return nil
		}
		return err
	}
}
