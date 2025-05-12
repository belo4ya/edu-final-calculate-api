package main

import (
	"context"
	"edu-final-calculate-api/internal/calculator/database"
	repository "edu-final-calculate-api/internal/calculator/repository/sqlite"
	"edu-final-calculate-api/internal/calculator/repository/sqlite/models"
	"fmt"

	"github.com/samber/lo"
)

func main() {
	ctx := context.Background()
	db := lo.Must(database.Connect(ctx, "./.data/db.sqlite"))
	repo := repository.New(db)

	//lo.Must0(repo.Register(ctx, models.RegisterUserCmd{
	//	Login:        "admin",
	//	PasswordHash: "admin",
	//}))

	user := lo.Must(repo.GetUser(ctx, models.GetUserCmd{
		Login:        "admin",
		PasswordHash: "admin",
	}))
	fmt.Println("user:", user)
}
