package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/auth"
	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/database"

	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/calc"
	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/config"
	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/repository"
	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/server"
	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/service"
	"github.com/belo4ya/edu-final-calculate-api/internal/logging"
	"github.com/belo4ya/edu-final-calculate-api/internal/mgmtserver"

	"github.com/belo4ya/runy"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	_ = godotenv.Load(".env.calculator")
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	ctx := runy.SetupSignalHandler()

	conf, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := logging.Configure(&logging.Config{Level: conf.LogLevel}); err != nil {
		return fmt.Errorf("configure logging: %w", err)
	}

	log := slog.Default()
	log.InfoContext(ctx, "logger is configured")
	log.InfoContext(ctx, "config initialized", "config", conf)

	auth_ := auth.New(conf)

	mgmtSrv := mgmtserver.New(&mgmtserver.Config{Addr: conf.MgmtAddr})
	grpcSrv := server.NewGRPCServer(conf, auth_)
	httpSrv := server.NewHTTPServer(conf)

	db, err := database.Connect(ctx, conf.DBSQLitePath)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}

	repo := repository.New(db)

	calcSvc := service.NewCalculatorService(conf, log, calc.NewCalculator(), repo)
	userSvc := service.NewUserService(conf, log, auth_, repo)
	agentSvc := service.NewAgentService(conf, log, repo)

	for i, svc := range []interface {
		RegisterWith(*grpc.Server)
		RegisterGRPCGateway(context.Context, *runtime.ServeMux, []grpc.DialOption) error
	}{calcSvc, userSvc, agentSvc} {
		svc.RegisterWith(grpcSrv.GRPC)
		if err := svc.RegisterGRPCGateway(ctx, httpSrv.GWMux, []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}); err != nil {
			return fmt.Errorf("register grpc gateway %d: %w", i, err)
		}
	}

	runy.Add(mgmtSrv, grpcSrv, httpSrv)
	if err := runy.Start(ctx); err != nil {
		return fmt.Errorf("problem with running app: %w", err)
	}
	return nil
}
