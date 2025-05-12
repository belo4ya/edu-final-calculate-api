package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/config"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Auth interface {
	UnaryServerInterceptor() grpc.UnaryServerInterceptor
}

type GRPCServer struct {
	GRPC    *grpc.Server
	conf    *config.Config
	metrics *grpcprom.ServerMetrics
}

func NewGRPCServer(conf *config.Config, auth Auth) *GRPCServer {
	srvMetrics := grpcprom.NewServerMetrics(grpcprom.WithServerHandlingTimeHistogram())
	prometheus.MustRegister(srvMetrics)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			srvMetrics.UnaryServerInterceptor(),
			grpcLoggingUnaryServerInterceptor(),
			auth.UnaryServerInterceptor(),
		),
	)
	reflection.Register(srv)

	return &GRPCServer{GRPC: srv, conf: conf, metrics: srvMetrics}
}

func (s *GRPCServer) Start(ctx context.Context) error {
	s.metrics.InitializeMetrics(s.GRPC)

	lis, err := net.Listen("tcp", s.conf.GRPCAddr)
	if err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		slog.InfoContext(ctx, "grpc server start listening on "+s.conf.GRPCAddr)
		if err := s.GRPC.Serve(lis); err != nil {
			errCh <- fmt.Errorf("start grpc server: %w", err)
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		slog.InfoContext(ctx, "shutting down grpc server")
		s.GRPC.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

func grpcLoggingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	interceptorLogger := func() logging.Logger {
		return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
			log := slog.With(fields...)
			switch lvl {
			case logging.LevelDebug:
				log.DebugContext(ctx, msg)
			case logging.LevelInfo:
				log.InfoContext(ctx, msg)
			case logging.LevelWarn:
				log.WarnContext(ctx, msg)
			case logging.LevelError:
				log.ErrorContext(ctx, msg)
			default: // should not happen
				panic(fmt.Sprintf("unknown level %v", lvl))
			}
		})
	}
	return logging.UnaryServerInterceptor(interceptorLogger())
}
