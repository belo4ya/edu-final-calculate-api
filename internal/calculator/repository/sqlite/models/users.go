package models

import "errors"

var (
	ErrUserExists   = errors.New("user exists")
	ErrUserNotFound = errors.New("user not found")
)

type User struct {
	ID           string `db:"id"`
	Login        string `db:"login"`
	PasswordHash string `db:"password_hash"`
}

type RegisterUserCmd struct {
	Login        string
	PasswordHash string
}

type GetUserCmd struct {
	Login        string
	PasswordHash string
}
