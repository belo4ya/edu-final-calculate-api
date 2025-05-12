package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/auth"

	calctypes "github.com/belo4ya/edu-final-calculate-api/internal/calculator/calc/types"
	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/config"
	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/repository/models"
	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/server"
	"github.com/belo4ya/edu-final-calculate-api/internal/logging"

	calculatorv1 "github.com/belo4ya/edu-final-calculate-api/pkg/calculator/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	Calculator interface {
		Parse(string) ([]calctypes.Token, error)
		Schedule([]calctypes.Token) []calctypes.Task
	}

	CalculatorRepository interface {
		CreateExpression(context.Context, string, models.CreateExpressionCmd) (string, error)
		ListExpressions(context.Context, string) ([]models.Expression, error)
		GetExpression(context.Context, string, string) (*models.Expression, error)
		ListExpressionTasks(context.Context, string, string) ([]models.Task, error)
	}
)

type CalculatorService struct {
	calculatorv1.UnimplementedCalculatorServiceServer
	conf *config.Config
	log  *slog.Logger
	calc Calculator
	repo CalculatorRepository
}

func NewCalculatorService(conf *config.Config, log *slog.Logger, calc Calculator, repo CalculatorRepository) *CalculatorService {
	return &CalculatorService{
		conf: conf,
		log:  logging.WithName(log, "calculator-service"),
		calc: calc,
		repo: repo,
	}
}

func (s *CalculatorService) RegisterWith(srv *grpc.Server) {
	calculatorv1.RegisterCalculatorServiceServer(srv, s)
}

func (s *CalculatorService) RegisterGRPCGateway(ctx context.Context, mux *runtime.ServeMux, clientOpts []grpc.DialOption) error {
	return calculatorv1.RegisterCalculatorServiceHandlerFromEndpoint(ctx, mux, "localhost"+s.conf.GRPCAddr, clientOpts)
}

func (s *CalculatorService) Calculate(
	ctx context.Context,
	req *calculatorv1.CalculateRequest,
) (*calculatorv1.CalculateResponse, error) {
	parsed, err := s.calc.Parse(req.Expression)
	if err != nil {
		if errors.Is(err, calctypes.ErrInvalidExpr) {
			server.WithHTTPResponseCode(ctx, http.StatusUnprocessableEntity)
			return nil, status.Error(codes.InvalidArgument, "invalid expression")
		}
		return nil, InternalError(fmt.Errorf("parse expression: %w", err))
	}

	tasks := s.calc.Schedule(parsed)

	createExpr := models.CreateExpressionCmd{
		Expression: req.Expression,
		Tasks:      make([]models.CreateExpressionCmdTask, 0, len(tasks)),
	}
	for _, t := range tasks {
		createExpr.Tasks = append(createExpr.Tasks, models.CreateExpressionCmdTask{
			ID:            t.ID,
			ParentTask1ID: t.ParentTask1ID,
			ParentTask2ID: t.ParentTask2ID,
			Arg1:          t.Arg1,
			Arg2:          t.Arg2,
			Operation:     s.mapTaskOperation(t.Operation),
			OperationTime: s.getTaskOperationTime(t.Operation),
		})
	}

	id, err := s.repo.CreateExpression(ctx, auth.MustUserIDFromContext(ctx), createExpr)
	if err != nil {
		return nil, InternalError(fmt.Errorf("create expression: %w", err))
	}

	server.WithHTTPResponseCode(ctx, http.StatusCreated)
	return &calculatorv1.CalculateResponse{Id: id}, nil
}

func (s *CalculatorService) ListExpressions(ctx context.Context, _ *emptypb.Empty) (*calculatorv1.ListExpressionsResponse, error) {
	exprs, err := s.repo.ListExpressions(ctx, auth.MustUserIDFromContext(ctx))
	if err != nil {
		return nil, InternalError(fmt.Errorf("list expressions: %w", err))
	}

	resp := &calculatorv1.ListExpressionsResponse{Expressions: make([]*calculatorv1.Expression, 0, len(exprs))}
	for _, expr := range exprs {
		resp.Expressions = append(resp.Expressions, mapExpressionToExpressionResponse(&expr))
	}
	return resp, nil
}

func (s *CalculatorService) GetExpression(
	ctx context.Context,
	req *calculatorv1.GetExpressionRequest,
) (*calculatorv1.GetExpressionResponse, error) {
	expr, err := s.repo.GetExpression(ctx, auth.MustUserIDFromContext(ctx), req.Id)
	if err != nil {
		if errors.Is(err, models.ErrExpressionNotFound) {
			return nil, status.Error(codes.NotFound, "expression not found")
		}
		return nil, InternalError(fmt.Errorf("get expression: %w", err))
	}

	return &calculatorv1.GetExpressionResponse{
		Expression: mapExpressionToExpressionResponse(expr),
	}, nil
}

func (s *CalculatorService) ListExpressionTasks(
	ctx context.Context,
	req *calculatorv1.ListExpressionTasksRequest,
) (*calculatorv1.ListExpressionTasksResponse, error) {
	tasks, err := s.repo.ListExpressionTasks(ctx, auth.MustUserIDFromContext(ctx), req.Id)
	if err != nil {
		if errors.Is(err, models.ErrExpressionNotFound) {
			return nil, status.Error(codes.NotFound, "expression not found")
		}
		return nil, InternalError(fmt.Errorf("list expression tasks: %w", err))
	}

	resp := &calculatorv1.ListExpressionTasksResponse{
		Tasks: make([]*calculatorv1.ListExpressionTasksResponse_Task, 0, len(tasks)),
	}
	for _, task := range tasks {
		resp.Tasks = append(resp.Tasks, mapTaskToInternalTaskResponse(task))
	}
	return resp, nil
}

func (s *CalculatorService) mapTaskOperation(op string) models.TaskOperation {
	switch op {
	case "+":
		return models.TaskOperationAddition
	case "-":
		return models.TaskOperationSubtraction
	case "*":
		return models.TaskOperationMultiplication
	case "/":
		return models.TaskOperationDivision
	default:
		return ""
	}
}

func (s *CalculatorService) getTaskOperationTime(op string) time.Duration {
	ms := 0
	switch op {
	case "+":
		ms = s.conf.TimeAdditionMs
	case "-":
		ms = s.conf.TimeSubtractionMs
	case "*":
		ms = s.conf.TimeMultiplicationMs
	case "/":
		ms = s.conf.TimeDivisionMs
	}
	return time.Duration(ms) * time.Millisecond
}
