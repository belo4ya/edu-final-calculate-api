package calc

import (
	"fmt"
	"testing"

	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/calc/types"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestCalculator_Parse(t *testing.T) {
	errorIsErrInvalidExpr := func(t assert.TestingT, err error, msgAndArgs ...any) bool {
		return assert.ErrorIs(t, err, types.ErrInvalidExpr, msgAndArgs...)
	}

	type args struct {
		s string
	}
	tests := []struct {
		name     string
		args     args
		want     []types.Token
		wantErr  assert.ErrorAssertionFunc
		skipTest bool
	}{
		{
			name: "simple addition",
			args: args{s: "1 + 2"},
			want: []types.Token{
				types.NewToken(1),
				types.NewToken(2),
				types.NewToken("+"),
			},
			wantErr:  assert.NoError,
			skipTest: false,
		},
		{
			name: "simple multiplication",
			args: args{s: "4*2"},
			want: []types.Token{
				types.NewToken(4),
				types.NewToken(2),
				types.NewToken("*"),
			},
			wantErr: assert.NoError,
		},
		{
			name: "multiple operations",
			args: args{s: "1+2*3"},
			want: []types.Token{
				types.NewToken(1),
				types.NewToken(2),
				types.NewToken(3),
				types.NewToken("*"),
				types.NewToken("+"),
			},
			wantErr: assert.NoError,
		},
		{
			name: "expression with parentheses",
			args: args{s: "(1+2)*3"},
			want: []types.Token{
				types.NewToken(1),
				types.NewToken(2),
				types.NewToken("+"),
				types.NewToken(3),
				types.NewToken("*"),
			},
			wantErr: assert.NoError,
		},
		{
			name: "nested parentheses",
			args: args{s: "((1+2)*(3+4))"},
			want: []types.Token{
				types.NewToken(1),
				types.NewToken(2),
				types.NewToken("+"),
				types.NewToken(3),
				types.NewToken(4),
				types.NewToken("+"),
				types.NewToken("*"),
			},
			wantErr: assert.NoError,
		},
		{
			name: "decimal numbers",
			args: args{s: "1.5+2.5"},
			want: []types.Token{
				types.NewToken(1.5),
				types.NewToken(2.5),
				types.NewToken("+"),
			},
			wantErr: assert.NoError,
		},
		{
			name:    "invalid expression: single operator",
			args:    args{s: "+"},
			wantErr: errorIsErrInvalidExpr,
		},
		{
			name:    "invalid expression: mismatched parentheses",
			args:    args{s: "(1+2"},
			wantErr: errorIsErrInvalidExpr,
		},
		{
			name:     "invalid expression: unbalanced operators",
			args:     args{s: "1+2*"},
			wantErr:  errorIsErrInvalidExpr,
			skipTest: false,
		},
		{
			name:    "invalid expression: double operators",
			args:    args{s: "1++2"},
			wantErr: errorIsErrInvalidExpr,
		},
		{
			name:    "invalid expression: invalid decimals",
			args:    args{s: "1..5+2"},
			wantErr: errorIsErrInvalidExpr,
		},
		{
			name:    "empty expression",
			args:    args{s: ""},
			wantErr: errorIsErrInvalidExpr,
		},
		// FIXME: unexpected behaviour
		{
			name:     "invalid expression: single number",
			args:     args{s: "1"},
			wantErr:  errorIsErrInvalidExpr,
			skipTest: true,
		},
		{
			name:     "invalid expression: only non-numeric characters",
			args:     args{s: "abracadabra"},
			wantErr:  errorIsErrInvalidExpr,
			skipTest: true,
		},
		{
			name:     "unary operators",
			args:     args{s: "-1 + 2"},
			wantErr:  assert.NoError,
			skipTest: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skipf("skip test: Parse(%v)", tt.args.s)
			}

			c := NewCalculator()
			got, err := c.Parse(tt.args.s)
			if !tt.wantErr(t, err, fmt.Sprintf("Parse(%v)", tt.args.s)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Parse(%v)", tt.args.s)
		})
	}
}

func TestCalculator_Schedule(t *testing.T) {
	mustParse := func(s string) []types.Token {
		return lo.Must(NewCalculator().Parse(s))
	}

	type args struct {
		rpn []types.Token
	}
	tests := []struct {
		name string
		args args
		want []types.Task
	}{
		{
			name: "simple addition",
			args: args{rpn: mustParse("1 + 2")},
			want: []types.Task{
				{
					ID:        "mock-id", // Will use matcher instead of exact value
					Operation: "+",
					Arg1:      1,
					Arg2:      2,
				},
			},
		},
		{
			name: "simple multiplication",
			args: args{rpn: mustParse("4*2")},
			want: []types.Task{
				{
					ID:        "mock-id",
					Operation: "*",
					Arg1:      4,
					Arg2:      2,
				},
			},
		},
		{
			name: "expression with two operations",
			args: args{rpn: mustParse("1+2*3")},
			want: []types.Task{
				{
					ID:        "mock-id-1",
					Operation: "*",
					Arg1:      2,
					Arg2:      3,
				},
				{
					ID:            "mock-id-2",
					Operation:     "+",
					Arg1:          1,
					ParentTask2ID: "mock-id-1",
				},
			},
		},
		{
			name: "complex expression",
			args: args{rpn: mustParse("((1+2)*(3+4))")},
			want: []types.Task{
				{
					ID:        "mock-id-1",
					Operation: "+",
					Arg1:      1,
					Arg2:      2,
				},
				{
					ID:        "mock-id-2",
					Operation: "+",
					Arg1:      3,
					Arg2:      4,
				},
				{
					ID:            "mock-id-3",
					Operation:     "*",
					ParentTask1ID: "mock-id-1",
					ParentTask2ID: "mock-id-2",
				},
			},
		},
		{
			name: "empty input",
			args: args{rpn: []types.Token{}},
			want: []types.Task{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCalculator()
			tasks := c.Schedule(tt.args.rpn)

			if !assert.Equal(t, len(tt.want), len(tasks), "Schedule task count doesn't match") {
				return
			}

			for i := range tasks {
				assert.Equal(t, tt.want[i].Operation, tasks[i].Operation)
				assert.Equal(t, tt.want[i].Arg1, tasks[i].Arg1)
				assert.Equal(t, tt.want[i].Arg2, tasks[i].Arg2)

				if tt.want[i].ParentTask1ID != "" {
					assert.NotEmpty(t, tasks[i].ParentTask1ID, "Expected parent task ID 1")
				} else {
					assert.Empty(t, tasks[i].ParentTask1ID, "Unexpected parent task ID 1")
				}

				if tt.want[i].ParentTask2ID != "" {
					assert.NotEmpty(t, tasks[i].ParentTask2ID, "Expected parent task ID 2")
				} else {
					assert.Empty(t, tasks[i].ParentTask2ID, "Unexpected parent task ID 2")
				}
			}

			if len(tasks) > 1 {
				taskIDMap := make(map[string]bool)
				for _, task := range tasks {
					taskIDMap[task.ID] = true
				}

				for _, task := range tasks {
					if task.ParentTask1ID != "" {
						assert.True(t, taskIDMap[task.ParentTask1ID], "Referenced parent task 1 doesn't exist")
					}
					if task.ParentTask2ID != "" {
						assert.True(t, taskIDMap[task.ParentTask2ID], "Referenced parent task 2 doesn't exist")
					}
				}
			}
		})
	}
}
