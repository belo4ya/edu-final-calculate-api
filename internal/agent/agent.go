package agent

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/belo4ya/edu-final-calculate-api/internal/agent/client"
	"github.com/belo4ya/edu-final-calculate-api/internal/agent/config"
	"github.com/belo4ya/edu-final-calculate-api/internal/logging"

	calculatorv1 "github.com/belo4ya/edu-final-calculate-api/pkg/calculator/v1"

	"github.com/avast/retry-go/v4"
)

type CalculatorAgentAPIClient interface {
	GetTask(ctx context.Context) (*calculatorv1.Task, error)
	SubmitTaskResult(ctx context.Context, res *calculatorv1.SubmitTaskResultRequest) error
}

// Agent is a worker that fetches and processes calculator tasks from a remote API.
// It implements a worker pool pattern to handle multiple tasks concurrently.
type Agent struct {
	conf   *config.Config
	log    *slog.Logger
	client CalculatorAgentAPIClient
}

// New creates a new Agent with the provided configuration, logger, and API client.
func New(conf *config.Config, log *slog.Logger, c CalculatorAgentAPIClient) *Agent {
	return &Agent{
		conf:   conf,
		log:    logging.WithName(log, "agent"),
		client: c,
	}
}

// Start launches the agent's worker pool based on configured computing power.
// It blocks until the context is canceled.
func (a *Agent) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < a.conf.ComputingPower; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.worker(ctx, i)
		}()
	}

	wg.Wait()
	return nil
}

// worker runs a continuous loop that fetches, executes, and submits results for calculator tasks.
// It will keep running until the context is canceled.
func (a *Agent) worker(ctx context.Context, workerID int) {
	log := a.log.With("worker_id", workerID)
	log.InfoContext(ctx, "worker started")

	for {
		select {
		case <-ctx.Done():
			log.InfoContext(ctx, "worker stopped")
			return
		default:
			task, err := a.fetchTask(ctx, log)
			if err != nil {
				continue // context done
			}

			log := log.With("task_id", task.Id)
			log.DebugContext(ctx, "executing task")

			result, err := a.executeTask(ctx, task)
			if err != nil {
				continue // context done
			}

			if err := a.submitTaskResult(ctx, log, task.Id, result); err != nil {
				continue // context done
			}

			log.InfoContext(ctx, "task completed", "result", result)
		}
	}
}

// executeTask performs the actual mathematical operation specified by the task.
// It simulates computation time by waiting for the duration specified in the task.
func (a *Agent) executeTask(ctx context.Context, task *calculatorv1.Task) (float64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-time.After(task.OperationTime.AsDuration()):
	}

	switch task.Operation {
	case calculatorv1.TaskOperation_TASK_OPERATION_ADDITION:
		return task.Arg1 + task.Arg2, nil
	case calculatorv1.TaskOperation_TASK_OPERATION_SUBTRACTION:
		return task.Arg1 - task.Arg2, nil
	case calculatorv1.TaskOperation_TASK_OPERATION_MULTIPLICATION:
		return task.Arg1 * task.Arg2, nil
	case calculatorv1.TaskOperation_TASK_OPERATION_DIVISION:
		if task.Arg2 == 0 {
			return math.NaN(), nil
		}
		return task.Arg1 / task.Arg2, nil
	default:
		return math.NaN(), nil
	}
}

// fetchTask retrieves a pending task from the remote API with exponential backoff.
// It will retry indefinitely until the context is canceled or a task is obtained.
func (a *Agent) fetchTask(ctx context.Context, log *slog.Logger) (*calculatorv1.Task, error) {
	task, _ := retry.DoWithData(
		func() (*calculatorv1.Task, error) {
			return a.client.GetTask(ctx)
		},
		retry.OnRetry(func(attempt uint, err error) {
			if errors.Is(err, client.ErrNoTasks) {
				log.DebugContext(ctx, "no tasks")
			} else {
				log.ErrorContext(ctx, "failed to fetch task", "error", err, "attempt", attempt)
			}
		}),
		retry.Context(ctx),
		retry.UntilSucceeded(),
		retry.Delay(200*time.Millisecond),
		retry.MaxDelay(10*time.Second),
		retry.MaxJitter(1*time.Second),
	)
	return task, ctx.Err()
}

// submitTaskResult sends the computed result back to the API with exponential backoff.
// It will retry indefinitely until the context is canceled or the submission succeeds.
func (a *Agent) submitTaskResult(ctx context.Context, log *slog.Logger, taskID string, result float64) error {
	req := &calculatorv1.SubmitTaskResultRequest{
		Id:     taskID,
		Result: result,
	}
	_ = retry.Do(
		func() error {
			return a.client.SubmitTaskResult(ctx, req)
		},
		retry.OnRetry(func(attempt uint, err error) {
			log.ErrorContext(ctx, "failed to submit task result", "error", err, "attempt", attempt)
		}),
		retry.Context(ctx),
		retry.UntilSucceeded(),
		retry.Delay(200*time.Millisecond),
		retry.MaxDelay(10*time.Second),
		retry.MaxJitter(1*time.Second),
	)
	return ctx.Err()
}
