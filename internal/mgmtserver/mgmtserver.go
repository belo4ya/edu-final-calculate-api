package mgmtserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	Addr string
}

type MGMTServer struct {
	HTTP *http.Server
	conf *Config
}

func New(conf *Config) *MGMTServer {
	srv := &MGMTServer{conf: conf}
	srv.HTTP = &http.Server{
		Addr:    conf.Addr,
		Handler: srv.routes(),
	}
	return srv
}

func (s *MGMTServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		slog.InfoContext(ctx, "mgmt server start listening on "+s.conf.Addr)
		if err := s.HTTP.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("start http server: %w", err)
		}
		close(errCh)
	}()
	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		slog.InfoContext(ctx, "shutting down mgmt server")
		if err := s.HTTP.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *MGMTServer) routes() *chi.Mux {
	okHandler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}

	r := chi.NewRouter()
	r.Handle("/metrics", promhttp.Handler())
	r.Mount("/debug", middleware.Profiler())
	r.Get("/healthz", okHandler)
	r.Get("/readyz", okHandler)
	return r
}
