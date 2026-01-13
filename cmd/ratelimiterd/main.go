package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cogni/internal/api"
	"cogni/internal/backend"
	"cogni/internal/backend/memory"
	"cogni/internal/backend/tb"
	"cogni/internal/registry"
	"cogni/pkg/ratelimiter"
)

// main launches ratelimiterd.
func main() {
	os.Exit(run())
}

// run executes ratelimiterd and returns an exit code.
func run() int {
	configPath := flag.String("config", "config.yaml", "path to ratelimiterd config")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return 1
	}

	reg := registry.New()
	if err := reg.Load(cfg.Registry.Path); err != nil {
		fmt.Fprintf(os.Stderr, "registry load error: %v\n", err)
		return 1
	}

	var limiter backend.Backend
	var closeBackend func()
	switch cfg.Server.Backend {
	case "tigerbeetle":
		clusterID, err := parseClusterID(cfg.TigerBeetle.ClusterID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cluster id error: %v\n", err)
			return 1
		}
		tbBackend, err := tb.New(tb.Config{
			ClusterID:      clusterID,
			Addresses:      cfg.TigerBeetle.Addresses,
			Sessions:       cfg.TigerBeetle.Sessions,
			MaxBatchEvents: cfg.TigerBeetle.MaxBatchEvents,
			FlushInterval:  flushInterval(cfg),
			Registry:       reg,
			RegistryPath:   cfg.Registry.Path,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "tb backend error: %v\n", err)
			return 1
		}
		limiter = tbBackend
		closeBackend = func() {
			_ = tbBackend.Close()
		}
	default:
		memBackend := memory.New(nil)
		memBackend.AttachRegistry(reg, cfg.Registry.Path)
		if err := applyStates(memBackend, reg.List()); err != nil {
			fmt.Fprintf(os.Stderr, "memory backend load error: %v\n", err)
			return 1
		}
		limiter = memBackend
	}

	handler := api.NewHandler(api.Config{
		Registry:     reg,
		Backend:      limiter,
		RegistryPath: cfg.Registry.Path,
		Now:          time.Now,
	})
	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	mux.Handle("/", handler)

	server := &http.Server{
		Addr:    cfg.Server.ListenAddr,
		Handler: mux,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	if closeBackend != nil {
		closeBackend()
	}
	return 0
}

// stateApplier loads persisted limit states into a backend.
type stateApplier interface {
	ApplyState(state ratelimiter.LimitState) error
}

// applyStates hydrates backends that support state loading.
func applyStates(b backend.Backend, states []ratelimiter.LimitState) error {
	applier, ok := b.(stateApplier)
	if !ok {
		return nil
	}
	for _, state := range states {
		if err := applier.ApplyState(state); err != nil {
			return err
		}
	}
	return nil
}
