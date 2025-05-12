package service

import (
	"context"
	"edu-final-calculate-api/internal/calculator/database/sqlz"
	"fmt"
	"math"
	"testing"
	"time"

	"edu-final-calculate-api/internal/calculator/config"
	"edu-final-calculate-api/internal/calculator/repository/models"
	"edu-final-calculate-api/internal/testutil"
	mocks "edu-final-calculate-api/internal/testutil/mocks/calculator/service"

	calculatorv1 "edu-final-calculate-api/pkg/calculator/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestAgentService_GetTask(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(repo *mocks.MockAgentRepository)
		want       *calculatorv1.GetTaskResponse
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name: "successfully retrieve pending task",
			setupMocks: func(repo *mocks.MockAgentRepository) {
				repo.EXPECT().GetPendingTask(mock.Anything).Return(&models.Task{
					ID:            "task1",
					ExpressionID:  "expr1",
					ParentTask1ID: sqlz.Some("parent1"),
					ParentTask2ID: sqlz.Some("parent2"),
					Arg1:          sqlz.Some[float64](5),
					Arg2:          sqlz.Some[float64](3),
					Operation:     models.TaskOperationAddition,
					OperationTime: time.Second,
					Status:        models.TaskStatusPending,
				}, nil)
			},
			want: &calculatorv1.GetTaskResponse{
				Task: &calculatorv1.Task{
					Id:            "task1",
					Arg1:          5,
					Arg2:          3,
					Operation:     calculatorv1.TaskOperation_TASK_OPERATION_ADDITION,
					OperationTime: durationpb.New(time.Second),
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "no pending tasks",
			setupMocks: func(repo *mocks.MockAgentRepository) {
				repo.EXPECT().GetPendingTask(mock.Anything).Return(nil, models.ErrNoPendingTasks)
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "repository error",
			setupMocks: func(repo *mocks.MockAgentRepository) {
				repo.EXPECT().GetPendingTask(mock.Anything).Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := mocks.NewMockAgentRepository(t)

			tt.setupMocks(repo)
			svc := NewAgentService(&config.Config{}, testutil.DiscardLogger(), repo)

			got, err := svc.GetTask(ctx, &emptypb.Empty{})
			if !tt.wantErr(t, err, fmt.Sprintf("GetTask(%v, %v)", ctx, &emptypb.Empty{})) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetTask(%v, %v)", ctx, &emptypb.Empty{})
		})
	}
}

func TestAgentService_SubmitTaskResult(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(repo *mocks.MockAgentRepository)
		req        *calculatorv1.SubmitTaskResultRequest
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name: "successfully submit completed task result",
			setupMocks: func(repo *mocks.MockAgentRepository) {
				repo.EXPECT().FinishTask(mock.Anything, models.FinishTaskCmd{
					ID:     "task1",
					Status: models.TaskStatusCompleted,
					Result: 42.0,
				}).Return(nil)
			},
			req: &calculatorv1.SubmitTaskResultRequest{
				Id:     "task1",
				Result: 42.0,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successfully submit failed task result",
			setupMocks: func(repo *mocks.MockAgentRepository) {
				repo.EXPECT().FinishTask(mock.Anything, models.FinishTaskCmd{
					ID:     "task1",
					Status: models.TaskStatusFailed,
					Result: 0,
				}).Return(nil)
			},
			req: &calculatorv1.SubmitTaskResultRequest{
				Id:     "task1",
				Result: math.NaN(),
			},
			wantErr: assert.NoError,
		},
		{
			name: "task not found",
			setupMocks: func(repo *mocks.MockAgentRepository) {
				repo.EXPECT().FinishTask(mock.Anything, mock.Anything).Return(models.ErrTaskNotFound)
			},
			req: &calculatorv1.SubmitTaskResultRequest{
				Id:     "nonexistent",
				Result: 10.0,
			},
			wantErr: assert.Error,
		},
		{
			name: "repository error",
			setupMocks: func(repo *mocks.MockAgentRepository) {
				repo.EXPECT().FinishTask(mock.Anything, mock.Anything).Return(assert.AnError)
			},
			req: &calculatorv1.SubmitTaskResultRequest{
				Id:     "task1",
				Result: 10.0,
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := mocks.NewMockAgentRepository(t)

			tt.setupMocks(repo)
			svc := NewAgentService(&config.Config{}, testutil.DiscardLogger(), repo)

			_, err := svc.SubmitTaskResult(ctx, tt.req)
			tt.wantErr(t, err, fmt.Sprintf("SubmitTaskResult(%v, %v)", ctx, tt.req))
		})
	}
}
