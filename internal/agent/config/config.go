package config

import (
	"encoding/json"
	"fmt"
	"reflect"

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
		CalculatorAPIAddr: "localhost:50051",
		ComputingPower:    4,
	}
	if err := env.Parse(conf); err != nil {
		return nil, fmt.Errorf("env parse: %w", err)
	}
	return conf, nil
}

func (c *Config) String() string {
	t, v := reflect.TypeOf(*c), reflect.ValueOf(*c)

	values := make(map[string]any)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		_, ok := field.Tag.Lookup("secret")
		if ok {
			values[field.Name] = "[SECRET]"
		} else {
			values[field.Name] = v.Field(i).Interface()
		}
	}

	bytes, err := json.Marshal(values)
	if err != nil {
		return fmt.Sprintf("json marshal: %v", err)
	}
	return string(bytes)
}
