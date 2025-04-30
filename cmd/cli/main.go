package main

import (
	"fmt"
	"log/slog"

	"edu-final-calculate-api/internal/calculator/calc"
)

func main() {
	s := "(2 + 2) + (1 + 1) * 3"

	c := &calc.Calculator{}
	rpn, err := c.Parse(s)
	if err != nil {
		slog.Error("error", "error", err)
	}
	fmt.Printf("rpn: %+v\n", rpn)

	fmt.Printf("plan: %+v\n", c.Schedule(rpn))
}
