package client

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/belo4ya/edu-final-calculate-api/internal/agent/config"

	calculatorv1 "github.com/belo4ya/edu-final-calculate-api/pkg/calculator/v1"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var ErrNoTasks = fmt.Errorf("no tasks")

type AgentAPI struct {
	client calculatorv1.AgentServiceClient
}

func NewAgentAPI(ctx context.Context, conf *config.Config) (*AgentAPI, func(), error) {
	clientMetrics := grpcprom.NewClientMetrics(grpcprom.WithClientHandlingTimeHistogram())
	prometheus.MustRegister(clientMetrics)

	conn, err := grpc.NewClient(
		conf.CalculatorAPIAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			timeout.UnaryClientInterceptor(10*time.Second),
			retry.UnaryClientInterceptor(
				retry.WithMax(3),
				retry.WithBackoff(retry.BackoffExponentialWithJitter(200*time.Millisecond, 0.1)),
			),
			clientMetrics.UnaryClientInterceptor(),
		),
	)
	if err != nil {
		return nil, func() {}, fmt.Errorf("init grpc client: %w", err)
	}

	cleanup := func() {
		if err := conn.Close(); err != nil {
			slog.ErrorContext(ctx, "failed to close client connection", "error", err)
		}
	}

	return &AgentAPI{client: calculatorv1.NewAgentServiceClient(conn)}, cleanup, nil
}

func (c *AgentAPI) GetTask(ctx context.Context) (*calculatorv1.Task, error) {
	resp, err := c.client.GetTask(ctx, nil)
	if err != nil {
		grpcStatus := status.Convert(err)
		if grpcStatus.Code() == codes.NotFound {
			return nil, ErrNoTasks
		}
		return nil, fmt.Errorf("get task: %w", err)
	}
	return resp.GetTask(), nil
}

func (c *AgentAPI) SubmitTaskResult(ctx context.Context, res *calculatorv1.SubmitTaskResultRequest) error {
	_, err := c.client.SubmitTaskResult(ctx, res)
	if err != nil {
		return fmt.Errorf("submit task result: %w", err)
	}
	return nil
}
