package types

import (
	"errors"
)

var ErrInvalidExpr = errors.New("invalid expression")

type Token struct {
	IsNumber bool
	Number   float64
	Symbol   string
}

func NewToken[T float64 | int | string](val T) Token {
	switch v := any(val).(type) {
	case float64:
		return Token{IsNumber: true, Number: v}
	case int:
		return Token{IsNumber: true, Number: float64(v)}
	case string:
		return Token{IsNumber: false, Symbol: v}
	default:
		panic("should not happen")
	}
}

type Task struct {
	ID            string
	ParentTask1ID string
	ParentTask2ID string

	Arg1      float64
	Arg2      float64
	Operation string
}
