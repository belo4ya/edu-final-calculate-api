package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"

	"edu-final-calculate-api/internal/calculator/config"
	"edu-final-calculate-api/internal/calculator/repository/models"
	"edu-final-calculate-api/internal/logging"
	calculatorv1 "edu-final-calculate-api/pkg/calculator/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AgentRepository interface {
	GetPendingTask(context.Context) (models.Task, error)
	FinishTask(context.Context, models.FinishTaskCmd) error
}

type AgentService struct {
	calculatorv1.UnimplementedAgentServiceServer
	conf *config.Config
	log  *slog.Logger
	repo AgentRepository
}

func NewAgentService(conf *config.Config, log *slog.Logger, repo AgentRepository) *AgentService {
	return &AgentService{
		conf: conf,
		log:  logging.WithName(log, "agent-service"),
		repo: repo,
	}
}

func (s *AgentService) RegisterWith(srv *grpc.Server) {
	calculatorv1.RegisterAgentServiceServer(srv, s)
}

func (s *AgentService) RegisterGRPCGateway(ctx context.Context, mux *runtime.ServeMux, clientOpts []grpc.DialOption) error {
	return calculatorv1.RegisterAgentServiceHandlerFromEndpoint(ctx, mux, "localhost"+s.conf.GRPCAddr, clientOpts)
}

func (s *AgentService) GetTask(ctx context.Context, _ *emptypb.Empty) (*calculatorv1.GetTaskResponse, error) {
	task, err := s.repo.GetPendingTask(ctx)
	if err != nil {
		if errors.Is(err, models.ErrNoPendingTasks) {
			return nil, status.Error(codes.NotFound, "no pending tasks")
		}
		return nil, InternalError(fmt.Errorf("get pending task: %w", err))
	}

	return &calculatorv1.GetTaskResponse{
		Task: mapTaskToAgentTaskResponse(task),
	}, nil
}

func (s *AgentService) SubmitTaskResult(ctx context.Context, req *calculatorv1.SubmitTaskResultRequest) (*emptypb.Empty, error) {
	var finishTaskCmd models.FinishTaskCmd
	if math.IsNaN(req.Result) {
		finishTaskCmd = models.FinishTaskCmd{
			ID:     req.Id,
			Status: models.TaskStatusFailed,
			Result: 0,
		}
	} else {
		finishTaskCmd = models.FinishTaskCmd{
			ID:     req.Id,
			Status: models.TaskStatusCompleted,
			Result: req.Result,
		}
	}

	if err := s.repo.FinishTask(ctx, finishTaskCmd); err != nil {
		if errors.Is(err, models.ErrTaskNotFound) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, InternalError(fmt.Errorf("finish task: %w", err))
	}
	return &emptypb.Empty{}, nil
}
