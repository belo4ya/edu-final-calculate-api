package main

import (
	"edu-final-calculate-api/internal/calculator/config"
	"log/slog"
)

func main() {
	conf, err := config.Load()
	if err != nil {
		panic(err)
	}
	slog.Info("load config", "conf", conf.String())

	//ctx := context.Background()
	//_, err := database.Connect(ctx, "./.data/db.db")
	//if err != nil {
	//	panic(err)
	//}
}
