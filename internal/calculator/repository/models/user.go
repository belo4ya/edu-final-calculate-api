package models

import "errors"

var (
	ErrUserExists   = errors.New("user exists")
	ErrUserNotFound = errors.New("user not found")
)

type RegisterUserCmd struct {
	Login        string `json:"login"`
	PasswordHash string `json:"password_hash"`
}

type GetUserCmd struct {
	Login        string `json:"login"`
	PasswordHash string `json:"password_hash"`
}

type User struct {
	ID           string `json:"id"`
	Login        string `json:"login"`
	PasswordHash string `json:"password_hash"`
}
