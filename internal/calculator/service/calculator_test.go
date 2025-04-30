package service

import (
	"context"
	"fmt"
	"testing"

	calctypes "edu-final-calculate-api/internal/calculator/calc/types"
	"edu-final-calculate-api/internal/calculator/config"
	"edu-final-calculate-api/internal/calculator/repository/models"
	"edu-final-calculate-api/internal/testutil"
	mocks "edu-final-calculate-api/internal/testutil/mocks/calculator/service"

	calculatorv1 "edu-final-calculate-api/pkg/calculator/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestCalculatorService_Calculate(t *testing.T) {
	conf := &config.Config{
		TimeAdditionMs:       1000,
		TimeSubtractionMs:    1000,
		TimeMultiplicationMs: 1000,
		TimeDivisionMs:       1000,
	}

	type args struct {
		ctx context.Context
		req *calculatorv1.CalculateRequest
	}
	tests := []struct {
		name       string
		setupMocks func(calc *mocks.MockCalculator, repo *mocks.MockCalculatorRepository)
		args       args
		want       *calculatorv1.CalculateResponse
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name: "successful calculation",
			setupMocks: func(calc *mocks.MockCalculator, repo *mocks.MockCalculatorRepository) {
				calc.EXPECT().Parse("1+2*3").Return([]calctypes.Token{
					calctypes.NewToken(1),
					calctypes.NewToken(2),
					calctypes.NewToken(3),
					calctypes.NewToken("*"),
					calctypes.NewToken("+"),
				}, nil)

				calc.EXPECT().Schedule(mock.Anything).Return([]calctypes.Task{
					{ID: "task1", Arg1: 2, Arg2: 3, Operation: "*"},
					{ID: "task2", ParentTask1ID: "task1", Arg1: 1, Operation: "+"},
				})

				repo.EXPECT().CreateExpression(mock.Anything,
					models.CreateExpressionCmd{Expression: "1+2*3"},
					mock.MatchedBy(func(tasks []models.CreateExpressionTaskCmd) bool {
						return len(tasks) == 2
					})).Return("expr123", nil)
			},
			args: args{
				ctx: context.Background(),
				req: &calculatorv1.CalculateRequest{
					Expression: "1+2*3",
				},
			},
			want:    &calculatorv1.CalculateResponse{Id: "expr123"},
			wantErr: assert.NoError,
		},
		{
			name: "invalid expression",
			setupMocks: func(calc *mocks.MockCalculator, repo *mocks.MockCalculatorRepository) {
				calc.EXPECT().Parse("1++2").Return(nil, calctypes.ErrInvalidExpr)
			},
			args: args{
				ctx: context.Background(),
				req: &calculatorv1.CalculateRequest{
					Expression: "1++2",
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "parse error",
			setupMocks: func(calc *mocks.MockCalculator, repo *mocks.MockCalculatorRepository) {
				calc.EXPECT().Parse(mock.Anything).Return(nil, assert.AnError)
			},
			args: args{
				ctx: context.Background(),
				req: &calculatorv1.CalculateRequest{
					Expression: "1+2",
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "repository error",
			setupMocks: func(calc *mocks.MockCalculator, repo *mocks.MockCalculatorRepository) {
				calc.EXPECT().Parse("1+2").Return([]calctypes.Token{
					calctypes.NewToken("+"),
					calctypes.NewToken(1),
					calctypes.NewToken(2),
				}, nil)

				calc.EXPECT().Schedule(mock.Anything).Return([]calctypes.Task{
					{ID: "task1", Arg1: 1, Arg2: 2, Operation: "+"},
				})

				repo.EXPECT().CreateExpression(mock.Anything, mock.Anything, mock.Anything).
					Return("", assert.AnError)
			},
			args: args{
				ctx: context.Background(),
				req: &calculatorv1.CalculateRequest{
					Expression: "1+2",
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := mocks.NewMockCalculator(t)
			repo := mocks.NewMockCalculatorRepository(t)

			tt.setupMocks(calc, repo)
			svc := NewCalculatorService(conf, testutil.DiscardLogger(), calc, repo)

			got, err := svc.Calculate(tt.args.ctx, tt.args.req)
			if !tt.wantErr(t, err, fmt.Sprintf("Calculate(%v, %v)", tt.args.ctx, tt.args.req)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Calculate(%v, %v)", tt.args.ctx, tt.args.req)
		})
	}
}

func TestCalculatorService_ListExpressions(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(calc *mocks.MockCalculator, repo *mocks.MockCalculatorRepository)
		want       *calculatorv1.ListExpressionsResponse
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name: "successful listing with multiple expressions",
			setupMocks: func(_ *mocks.MockCalculator, repo *mocks.MockCalculatorRepository) {
				repo.EXPECT().ListExpressions(mock.Anything).Return([]models.Expression{
					{
						ID:         "expr1",
						Expression: "1+2",
						Status:     models.ExpressionStatusCompleted,
						Result:     3,
					},
					{
						ID:         "expr2",
						Expression: "3*4",
						Status:     models.ExpressionStatusInProgress,
					},
				}, nil)
			},
			want: &calculatorv1.ListExpressionsResponse{
				Expressions: []*calculatorv1.Expression{
					{
						Id:         "expr1",
						Expression: "1+2",
						Status:     calculatorv1.ExpressionStatus_EXPRESSION_STATUS_COMPLETED,
						Result:     3,
					},
					{
						Id:         "expr2",
						Expression: "3*4",
						Status:     calculatorv1.ExpressionStatus_EXPRESSION_STATUS_IN_PROGRESS,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "successful listing with empty result",
			setupMocks: func(_ *mocks.MockCalculator, repo *mocks.MockCalculatorRepository) {
				repo.EXPECT().ListExpressions(mock.Anything).Return([]models.Expression{}, nil)
			},
			want: &calculatorv1.ListExpressionsResponse{
				Expressions: []*calculatorv1.Expression{},
			},
			wantErr: assert.NoError,
		},
		{
			name: "repository error",
			setupMocks: func(_ *mocks.MockCalculator, repo *mocks.MockCalculatorRepository) {
				repo.EXPECT().ListExpressions(mock.Anything).Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			calc := mocks.NewMockCalculator(t)
			repo := mocks.NewMockCalculatorRepository(t)

			tt.setupMocks(calc, repo)
			svc := NewCalculatorService(&config.Config{}, testutil.DiscardLogger(), calc, repo)

			got, err := svc.ListExpressions(ctx, &emptypb.Empty{})
			if !tt.wantErr(t, err, fmt.Sprintf("ListExpressions(%v, %v)", ctx, &emptypb.Empty{})) {
				return
			}
			assert.Equalf(t, tt.want, got, "ListExpressions(%v, %v)", ctx, &emptypb.Empty{})
		})
	}
}

func TestCalculatorService_GetExpression(t *testing.T) {
	type args struct {
		req *calculatorv1.GetExpressionRequest
	}
	tests := []struct {
		name       string
		setupMocks func(calc *mocks.MockCalculator, repo *mocks.MockCalculatorRepository)
		args       args
		want       *calculatorv1.GetExpressionResponse
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name: "existing expression found",
			setupMocks: func(_ *mocks.MockCalculator, repo *mocks.MockCalculatorRepository) {
				repo.EXPECT().GetExpression(mock.Anything, "expr1").Return(models.Expression{
					ID:         "expr1",
					Expression: "1+2*3",
					Status:     models.ExpressionStatusCompleted,
					Result:     7,
				}, nil)
			},
			args: args{
				req: &calculatorv1.GetExpressionRequest{
					Id: "expr1",
				},
			},
			want: &calculatorv1.GetExpressionResponse{
				Expression: &calculatorv1.Expression{
					Id:         "expr1",
					Expression: "1+2*3",
					Status:     calculatorv1.ExpressionStatus_EXPRESSION_STATUS_COMPLETED,
					Result:     7,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "in-progress expression found",
			setupMocks: func(_ *mocks.MockCalculator, repo *mocks.MockCalculatorRepository) {
				repo.EXPECT().GetExpression(mock.Anything, "expr2").Return(models.Expression{
					ID:         "expr2",
					Expression: "5*8-3",
					Status:     models.ExpressionStatusInProgress,
				}, nil)
			},
			args: args{
				req: &calculatorv1.GetExpressionRequest{
					Id: "expr2",
				},
			},
			want: &calculatorv1.GetExpressionResponse{
				Expression: &calculatorv1.Expression{
					Id:         "expr2",
					Expression: "5*8-3",
					Status:     calculatorv1.ExpressionStatus_EXPRESSION_STATUS_IN_PROGRESS,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "expression not found",
			setupMocks: func(_ *mocks.MockCalculator, repo *mocks.MockCalculatorRepository) {
				repo.EXPECT().GetExpression(mock.Anything, "non-existent").Return(models.Expression{}, models.ErrExpressionNotFound)
			},
			args: args{
				req: &calculatorv1.GetExpressionRequest{
					Id: "non-existent",
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "repository error",
			setupMocks: func(_ *mocks.MockCalculator, repo *mocks.MockCalculatorRepository) {
				repo.EXPECT().GetExpression(mock.Anything, mock.Anything).Return(models.Expression{}, assert.AnError)
			},
			args: args{
				req: &calculatorv1.GetExpressionRequest{
					Id: "expr3",
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			calc := mocks.NewMockCalculator(t)
			repo := mocks.NewMockCalculatorRepository(t)

			tt.setupMocks(calc, repo)
			svc := NewCalculatorService(&config.Config{}, testutil.DiscardLogger(), calc, repo)

			got, err := svc.GetExpression(ctx, tt.args.req)
			if !tt.wantErr(t, err, fmt.Sprintf("GetExpression(%v, %v)", ctx, tt.args.req)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetExpression(%v, %v)", ctx, tt.args.req)
		})
	}
}
