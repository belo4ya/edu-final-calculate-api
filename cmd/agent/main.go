package main

import (
	"fmt"
	"log/slog"
	"os"

	"edu-final-calculate-api/internal/agent"
	"edu-final-calculate-api/internal/agent/client"
	"edu-final-calculate-api/internal/agent/config"
	"edu-final-calculate-api/internal/logging"
	"edu-final-calculate-api/internal/mgmtserver"

	"github.com/belo4ya/runy"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env.agent")
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

	calculatorClient, cleanup, err := client.NewAgentAPI(ctx, conf)
	if err != nil {
		return fmt.Errorf("create calculator client: %w", err)
	}
	defer cleanup()

	mgmtSrv := mgmtserver.New(&mgmtserver.Config{Addr: conf.MgmtAddr})

	_agent := agent.New(conf, log, calculatorClient)

	runy.Add(mgmtSrv, _agent)
	if err := runy.Start(ctx); err != nil {
		return fmt.Errorf("problem with running app: %w", err)
	}
	return nil
}
