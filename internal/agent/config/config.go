package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	LogLevel          string `env:"LOG_LEVEL"`
	MgmtAddr          string `env:"MGMT_ADDR"`
	CalculatorAPIAddr string `env:"CALCULATOR_API_ADDR"`
	ComputingPower    int    `env:"COMPUTING_POWER"`
}

func Load() (*Config, error) {
	conf := &Config{
		LogLevel:          "info",
		MgmtAddr:          ":8082",
		CalculatorAPIAddr: ":50051",
		ComputingPower:    4,
	}
	if err := env.Parse(conf); err != nil {
		return nil, fmt.Errorf("env parse: %w", err)
	}
	return conf, nil
}
