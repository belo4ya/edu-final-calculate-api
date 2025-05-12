package calc

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/calc/stackx"
	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/calc/types"

	"github.com/rs/xid"
)

// Calculator handles expression parsing and scheduling for mathematical operations.
type Calculator struct{}

// NewCalculator creates a new instance of Calculator.
func NewCalculator() *Calculator {
	return &Calculator{}
}

// Parse converts a string expression into a sequence of tokens in Reverse Polish Notation (RPN).
// Returns types.ErrInvalidExpr if the expression is invalid or cannot be parsed.
func (c *Calculator) Parse(s string) ([]types.Token, error) {
	tokens, err := c.tokenize(s)
	if err != nil {
		return nil, fmt.Errorf("tokenize: %w", err)
	}
	rpn, err := c.toRPN(tokens)
	if err != nil {
		return nil, fmt.Errorf("to RPN: %w", err)
	}
	return rpn, nil
}

// Schedule transforms RPN tokens into a sequence of executable tasks.
// Each task represents an operation that depends on either values or results of other tasks.
func (c *Calculator) Schedule(rpn []types.Token) []types.Task {
	plan := make([]types.Task, 0, len(rpn))

	type stackItem struct {
		IsTask bool
		TaskID string
		Value  float64
	}
	stack := stackx.New[stackItem]()

	for _, token := range rpn {
		if token.IsNumber {
			stack.Push(stackItem{IsTask: false, Value: token.Number})
			continue
		}

		task := types.Task{ID: xid.New().String(), Operation: token.Symbol}

		right, left := stack.SafePop(), stack.SafePop()
		if left.IsTask {
			task.ParentTask1ID = left.TaskID
		} else {
			task.Arg1 = left.Value
		}
		if right.IsTask {
			task.ParentTask2ID = right.TaskID
		} else {
			task.Arg2 = right.Value
		}

		plan = append(plan, task)
		stack.Push(stackItem{IsTask: true, TaskID: task.ID})
	}

	return plan
}

// tokenize breaks an input string into individual tokens (numbers and operators).
// Returns types.ErrInvalidExpr if the expression contains invalid numeric values.
func (c *Calculator) tokenize(s string) ([]types.Token, error) {
	tokens := make([]types.Token, 0, len(s))
	var numberBuf strings.Builder
	for _, ch := range strings.Split(s, "") {
		if ch >= "0" && ch <= "9" || ch == "." {
			numberBuf.WriteString(ch)
		} else if ch != " " {
			if numberBuf.Len() > 0 {
				num, err := strconv.ParseFloat(numberBuf.String(), 64)
				if err != nil {
					return nil, types.ErrInvalidExpr
				}
				tokens = append(tokens, types.NewToken(num))
				numberBuf.Reset()
			}
			tokens = append(tokens, types.NewToken(ch))
		}
	}
	if numberBuf.Len() > 0 {
		num, err := strconv.ParseFloat(numberBuf.String(), 64)
		if err != nil {
			return nil, types.ErrInvalidExpr
		}
		tokens = append(tokens, types.NewToken(num))
	}
	return tokens, nil
}

// toRPN converts a sequence of tokens to Reverse Polish Notation using the shunting-yard algorithm.
// Returns types.ErrInvalidExpr if the resulting RPN expression is invalid.
func (c *Calculator) toRPN(tokens []types.Token) ([]types.Token, error) {
	rpn := make([]types.Token, 0, len(tokens))
	stack := stackx.New[types.Token]()
	for _, t := range tokens {
		switch {
		case t.IsNumber:
			rpn = append(rpn, t)
		case t.Symbol == "(":
			stack.Push(t)
		case t.Symbol == ")":
			for stack.Size() > 0 && stack.SafePeek().Symbol != "(" {
				rpn = append(rpn, stack.SafePop())
			}
			if stack.Size() > 0 {
				stack.SafePop()
			}
		default:
			for stack.Size() > 0 && c.precedence(stack.SafePeek().Symbol) >= c.precedence(t.Symbol) {
				rpn = append(rpn, stack.SafePop())
			}
			stack.Push(t)
		}
	}
	for stack.Size() > 0 {
		rpn = append(rpn, stack.SafePop())
	}

	if err := c.validateRPN(rpn); err != nil {
		return nil, err
	}
	return rpn, nil
}

// validateRPN checks if the RPN expression is valid by simulating its evaluation.
// Returns an error if the expression is malformed or contains invalid operations.
func (c *Calculator) validateRPN(rpn []types.Token) error {
	stack := stackx.New[types.Token]()
	for _, token := range rpn {
		if !c.isOp(token.Symbol) {
			stack.Push(token)
			continue
		}

		if stack.Size() < 2 {
			return types.ErrInvalidExpr
		}

		_, _ = stack.SafePop(), stack.SafePop()
		stack.Push(types.NewToken("$"))
	}

	if stack.Size() != 1 {
		return types.ErrInvalidExpr
	}
	return nil
}

func (c *Calculator) precedence(op string) int {
	switch op {
	case "+", "-":
		return 1
	case "*", "/":
		return 2
	default:
		return 0
	}
}

func (c *Calculator) isOp(s string) bool {
	switch s {
	case "+", "-", "*", "/":
		return true
	default:
		return false
	}
}
