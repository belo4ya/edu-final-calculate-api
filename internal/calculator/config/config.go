package config

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	LogLevel     string `env:"LOG_LEVEL"`
	MgmtAddr     string `env:"MGMT_ADDR"`
	GRPCAddr     string `env:"GRPC_ADDR"`
	HTTPAddr     string `env:"HTTP_ADDR"`
	DBSQLitePath string `env:"DB_SQLITE_PATH"`

	AuthJWTSecret string `env:"AUTH_JWT_SECRET" secret:""`

	TimeAdditionMs       int `env:"TIME_ADDITION_MS"`
	TimeSubtractionMs    int `env:"TIME_SUBTRACTION_MS"`
	TimeMultiplicationMs int `env:"TIME_MULTIPLICATIONS_MS"`
	TimeDivisionMs       int `env:"TIME_DIVISIONS_MS"`
}

func Load() (*Config, error) {
	conf := &Config{
		LogLevel:             "info",
		MgmtAddr:             ":8081",
		GRPCAddr:             ":50051",
		HTTPAddr:             ":8080",
		DBSQLitePath:         ".data/db.sqlite",
		AuthJWTSecret:        "jwt-secret",
		TimeAdditionMs:       1000,
		TimeSubtractionMs:    1000,
		TimeMultiplicationMs: 1000,
		TimeDivisionMs:       1000,
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
