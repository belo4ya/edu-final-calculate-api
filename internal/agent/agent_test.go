package agent

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"edu-final-calculate-api/internal/agent/client"
	"edu-final-calculate-api/internal/agent/config"
	"edu-final-calculate-api/internal/testutil"
	mocks "edu-final-calculate-api/internal/testutil/mocks/agent"
	calculatorv1 "edu-final-calculate-api/pkg/calculator/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestAgent_executeTask(t *testing.T) {
	type args struct {
		ctx  context.Context
		task *calculatorv1.Task
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantNaN bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "addition operation",
			args: args{
				ctx: context.Background(),
				task: &calculatorv1.Task{
					Id:        "task1",
					Operation: calculatorv1.TaskOperation_TASK_OPERATION_ADDITION,
					Arg1:      5,
					Arg2:      3,
				},
			},
			want:    8,
			wantErr: assert.NoError,
		},
		{
			name: "subtraction operation",
			args: args{
				ctx: context.Background(),
				task: &calculatorv1.Task{
					Id:        "task2",
					Operation: calculatorv1.TaskOperation_TASK_OPERATION_SUBTRACTION,
					Arg1:      10,
					Arg2:      4,
				},
			},
			want:    6,
			wantErr: assert.NoError,
		},
		{
			name: "multiplication operation",
			args: args{
				ctx: context.Background(),
				task: &calculatorv1.Task{
					Id:        "task3",
					Operation: calculatorv1.TaskOperation_TASK_OPERATION_MULTIPLICATION,
					Arg1:      7,
					Arg2:      6,
				},
			},
			want:    42,
			wantErr: assert.NoError,
		},
		{
			name: "division operation",
			args: args{
				ctx: context.Background(),
				task: &calculatorv1.Task{
					Id:        "task4",
					Operation: calculatorv1.TaskOperation_TASK_OPERATION_DIVISION,
					Arg1:      20,
					Arg2:      5,
				},
			},
			want:    4,
			wantErr: assert.NoError,
		},
		{
			name: "division by zero",
			args: args{
				ctx: context.Background(),
				task: &calculatorv1.Task{
					Id:        "task5",
					Operation: calculatorv1.TaskOperation_TASK_OPERATION_DIVISION,
					Arg1:      10,
					Arg2:      0,
				},
			},
			wantNaN: true,
			wantErr: assert.NoError,
		},
		{
			name: "unknown operation",
			args: args{
				ctx: context.Background(),
				task: &calculatorv1.Task{
					Id:        "task6",
					Operation: calculatorv1.TaskOperation_TASK_OPERATION_UNSPECIFIED,
					Arg1:      5,
					Arg2:      5,
				},
			},
			wantNaN: true,
			wantErr: assert.NoError,
		},
		{
			name: "context canceled",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				task: &calculatorv1.Task{
					Id:            "task7",
					Operation:     calculatorv1.TaskOperation_TASK_OPERATION_ADDITION,
					OperationTime: durationpb.New(200 * time.Millisecond),
					Arg1:          1,
					Arg2:          1,
				},
			},
			want:    0,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := mocks.NewMockCalculatorAgentAPIClient(t)

			agent := New(&config.Config{}, testutil.DiscardLogger(), mc)

			got, err := agent.executeTask(tt.args.ctx, tt.args.task)
			if !tt.wantErr(t, err, fmt.Sprintf("executeTask(%v, %v)", tt.args.ctx, tt.args.task)) {
				return
			}
			if tt.wantNaN {
				assert.True(t, math.IsNaN(got), "executeTask(%v, %v)", tt.args.ctx, tt.args.task)
			} else {
				assert.Equalf(t, tt.want, got, "executeTask(%v, %v)", tt.args.ctx, tt.args.task)
			}
		})
	}
}

func TestAgent_fetchTask(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name       string
		setupMocks func(client *mocks.MockCalculatorAgentAPIClient)
		args       args
		want       *calculatorv1.Task
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name: "successful fetch",
			setupMocks: func(c *mocks.MockCalculatorAgentAPIClient) {
				c.EXPECT().GetTask(mock.Anything).Return(&calculatorv1.Task{
					Id:        "task1",
					Arg1:      10,
					Arg2:      5,
					Operation: calculatorv1.TaskOperation_TASK_OPERATION_ADDITION,
				}, nil).Once()
			},
			args: args{ctx: context.Background()},
			want: &calculatorv1.Task{
				Id:        "task1",
				Arg1:      10,
				Arg2:      5,
				Operation: calculatorv1.TaskOperation_TASK_OPERATION_ADDITION,
			},
			wantErr: assert.NoError,
		},
		{
			name: "context canceled",
			setupMocks: func(client *mocks.MockCalculatorAgentAPIClient) {
				client.EXPECT().GetTask(mock.Anything).Return(nil, context.Canceled).Maybe()
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "retry once then succeed",
			setupMocks: func(client *mocks.MockCalculatorAgentAPIClient) {
				client.EXPECT().GetTask(mock.Anything).Return(nil, assert.AnError).Once()
				client.EXPECT().GetTask(mock.Anything).Return(&calculatorv1.Task{
					Id:        "task2",
					Arg1:      7,
					Arg2:      8,
					Operation: calculatorv1.TaskOperation_TASK_OPERATION_MULTIPLICATION,
				}, nil).Once()
			},
			args: args{ctx: context.Background()},
			want: &calculatorv1.Task{
				Id:        "task2",
				Arg1:      7,
				Arg2:      8,
				Operation: calculatorv1.TaskOperation_TASK_OPERATION_MULTIPLICATION,
			},
			wantErr: assert.NoError,
		},
		{
			name: "no tasks available then succeed",
			setupMocks: func(c *mocks.MockCalculatorAgentAPIClient) {
				c.EXPECT().GetTask(mock.Anything).Return(nil, client.ErrNoTasks).Once()
				c.EXPECT().GetTask(mock.Anything).Return(&calculatorv1.Task{
					Id:        "task3",
					Arg1:      20,
					Arg2:      4,
					Operation: calculatorv1.TaskOperation_TASK_OPERATION_DIVISION,
				}, nil).Once()
			},
			args: args{ctx: context.Background()},
			want: &calculatorv1.Task{
				Id:        "task3",
				Arg1:      20,
				Arg2:      4,
				Operation: calculatorv1.TaskOperation_TASK_OPERATION_DIVISION,
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := testutil.DiscardLogger()
			mc := mocks.NewMockCalculatorAgentAPIClient(t)

			tt.setupMocks(mc)
			agent := New(&config.Config{}, log, mc)

			got, err := agent.fetchTask(tt.args.ctx, log)
			if !tt.wantErr(t, err, fmt.Sprintf("fetchTask(%v, %v)", tt.args.ctx, log)) {
				return
			}
			assert.Equalf(t, tt.want, got, "fetchTask(%v, %v)", tt.args.ctx, log)
		})
	}
}

func TestAgent_submitTaskResult(t *testing.T) {
	type args struct {
		ctx    context.Context
		taskID string
		result float64
	}
	tests := []struct {
		name       string
		setupMocks func(client *mocks.MockCalculatorAgentAPIClient)
		args       args
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name: "successful submission",
			setupMocks: func(c *mocks.MockCalculatorAgentAPIClient) {
				c.EXPECT().SubmitTaskResult(mock.Anything, &calculatorv1.SubmitTaskResultRequest{
					Id:     "task1",
					Result: 15,
				}).Return(nil).Once()
			},
			args: args{
				ctx:    context.Background(),
				taskID: "task1",
				result: 15,
			},
			wantErr: assert.NoError,
		},
		{
			name: "context canceled",
			setupMocks: func(client *mocks.MockCalculatorAgentAPIClient) {
				client.EXPECT().SubmitTaskResult(mock.Anything, mock.Anything).Return(context.Canceled).Maybe()
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				taskID: "task2",
				result: 25,
			},
			wantErr: assert.Error,
		},
		{
			name: "retry once then succeed",
			setupMocks: func(client *mocks.MockCalculatorAgentAPIClient) {
				req := &calculatorv1.SubmitTaskResultRequest{
					Id:     "task3",
					Result: 42,
				}
				client.EXPECT().SubmitTaskResult(mock.Anything, req).Return(assert.AnError).Once()
				client.EXPECT().SubmitTaskResult(mock.Anything, req).Return(nil).Once()
			},
			args: args{
				ctx:    context.Background(),
				taskID: "task3",
				result: 42,
			},
			wantErr: assert.NoError,
		},
		{
			name: "submit result with NaN",
			setupMocks: func(c *mocks.MockCalculatorAgentAPIClient) {
				c.EXPECT().SubmitTaskResult(mock.Anything, mock.MatchedBy(func(req *calculatorv1.SubmitTaskResultRequest) bool {
					return req.Id == "task4" && math.IsNaN(req.Result)
				})).Return(nil).Once()
			},
			args: args{
				ctx:    context.Background(),
				taskID: "task4",
				result: math.NaN(),
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := testutil.DiscardLogger()
			mc := mocks.NewMockCalculatorAgentAPIClient(t)

			tt.setupMocks(mc)
			agent := New(&config.Config{}, log, mc)

			tt.wantErr(
				t,
				agent.submitTaskResult(tt.args.ctx, log, tt.args.taskID, tt.args.result),
				fmt.Sprintf("submitTaskResult(%v, %v, %v, %v)", tt.args.ctx, log, tt.args.taskID, tt.args.result),
			)
		})
	}
}
