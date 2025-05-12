package repository

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/repository/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_CreateExpression(t *testing.T) {
	db := setupTestDB(t)
	repo := New(db)
	ctx := context.Background()

	userID := createTestUser(t, repo, ctx)

	tests := []struct {
		name    string
		cmd     models.CreateExpressionCmd
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "simple expression with one task",
			cmd: models.CreateExpressionCmd{
				Expression: "5+3",
				Tasks: []models.CreateExpressionCmdTask{
					{
						ID:            "task1",
						ParentTask1ID: "",
						ParentTask2ID: "",
						Arg1:          5,
						Arg2:          3,
						Operation:     models.TaskOperationAddition,
						OperationTime: time.Millisecond * 100,
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "complex expression with multiple tasks",
			cmd: models.CreateExpressionCmd{
				Expression: "(5+3)*2",
				Tasks: []models.CreateExpressionCmdTask{
					{
						ID:            "task2",
						ParentTask1ID: "",
						ParentTask2ID: "",
						Arg1:          5,
						Arg2:          3,
						Operation:     models.TaskOperationAddition,
						OperationTime: time.Millisecond * 100,
					},
					{
						ID:            "task3",
						ParentTask1ID: "task2",
						ParentTask2ID: "",
						Arg1:          0,
						Arg2:          2,
						Operation:     models.TaskOperationMultiplication,
						OperationTime: time.Millisecond * 50,
					},
				},
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exprID, err := repo.CreateExpression(ctx, userID, tt.cmd)
			tt.wantErr(t, err)
			assert.NotEmpty(t, exprID, "Expression ID should not be empty")
		})
	}
}

func TestRepository_ListExpressions(t *testing.T) {
	db := setupTestDB(t)
	repo := New(db)
	ctx := context.Background()

	userID := createTestUser(t, repo, ctx)
	otherUserID := createTestUser(t, repo, ctx)

	createTestExpressions(t, repo, ctx, userID, 3)
	createTestExpressions(t, repo, ctx, otherUserID, 2)

	// Test listing expressions for the user
	expressions, err := repo.ListExpressions(ctx, userID)
	require.NoError(t, err, "Failed to list expressions")

	assert.Len(t, expressions, 3, "Should return 3 expressions for the user")
	for _, expr := range expressions {
		assert.Equal(t, userID, expr.UserID, "Expression should belong to the correct user")
	}

	// Test listing expressions for the other user
	otherExpressions, err := repo.ListExpressions(ctx, otherUserID)
	require.NoError(t, err, "Failed to list expressions")

	assert.Len(t, otherExpressions, 2, "Should return 2 expressions for the other user")
	for _, expr := range otherExpressions {
		assert.Equal(t, otherUserID, expr.UserID, "Expression should belong to the correct user")
	}
}

func TestRepository_GetExpression(t *testing.T) {
	db := setupTestDB(t)
	repo := New(db)
	ctx := context.Background()

	userID := createTestUser(t, repo, ctx)
	exprIDs := createTestExpressions(t, repo, ctx, userID, 1)
	exprID := exprIDs[0]

	tests := []struct {
		name    string
		userID  string
		exprID  string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:   "get existing expression",
			userID: userID,
			exprID: exprID,
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.NoError(t, err)
			},
		},
		{
			name:   "expression not found",
			userID: userID,
			exprID: "nonexistent-id",
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.ErrorIs(t, err, models.ErrExpressionNotFound)
			},
		},
		{
			name:   "wrong user",
			userID: "wrong-user-id",
			exprID: exprID,
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.ErrorIs(t, err, models.ErrExpressionNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := repo.GetExpression(ctx, tt.userID, tt.exprID)
			tt.wantErr(t, err)

			if err == nil {
				assert.Equal(t, tt.exprID, expr.ID)
				assert.Equal(t, tt.userID, expr.UserID)
				assert.NotEmpty(t, expr.Expression)
				assert.Equal(t, models.ExpressionStatusPending, expr.Status)
			}
		})
	}
}

func TestRepository_ListExpressionTasks(t *testing.T) {
	db := setupTestDB(t)
	repo := New(db)
	ctx := context.Background()

	userID := createTestUser(t, repo, ctx)
	cmd := models.CreateExpressionCmd{
		Expression: "(5+3)*2",
		Tasks: []models.CreateExpressionCmdTask{
			{
				ID:            "task1",
				ParentTask1ID: "",
				ParentTask2ID: "",
				Arg1:          5,
				Arg2:          3,
				Operation:     models.TaskOperationAddition,
				OperationTime: time.Millisecond * 100,
			},
			{
				ID:            "task2",
				ParentTask1ID: "task1",
				ParentTask2ID: "",
				Arg1:          0,
				Arg2:          2,
				Operation:     models.TaskOperationMultiplication,
				OperationTime: time.Millisecond * 50,
			},
		},
	}

	exprID, err := repo.CreateExpression(ctx, userID, cmd)
	require.NoError(t, err, "Failed to create test expression")

	tests := []struct {
		name      string
		userID    string
		exprID    string
		wantTasks int
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "get tasks for existing expression",
			userID:    userID,
			exprID:    exprID,
			wantTasks: 2,
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.NoError(t, err)
			},
		},
		{
			name:      "expression not found",
			userID:    userID,
			exprID:    "nonexistent-id",
			wantTasks: 0,
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.ErrorIs(t, err, models.ErrExpressionNotFound)
			},
		},
		{
			name:      "wrong user",
			userID:    "wrong-user-id",
			exprID:    exprID,
			wantTasks: 0,
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.ErrorIs(t, err, models.ErrExpressionNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := repo.ListExpressionTasks(ctx, tt.userID, tt.exprID)
			tt.wantErr(t, err)

			if err == nil {
				assert.Len(t, tasks, tt.wantTasks)
				for _, task := range tasks {
					assert.Equal(t, tt.exprID, task.ExpressionID)
				}

				// Verify the first task is in pending status
				assert.Equal(t, models.TaskStatusPending, tasks[0].Status)

				// Verify the second task is in created status
				assert.Equal(t, models.TaskStatusCreated, tasks[1].Status)

				// Verify parent relationship
				assert.Equal(t, "task1", tasks[1].ParentTask1ID.V)
			}
		})
	}
}

func TestRepository_GetPendingTask(t *testing.T) {
	db := setupTestDB(t)
	repo := New(db)
	ctx := context.Background()

	task, err := repo.GetPendingTask(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, models.ErrNoPendingTasks)
	require.Nil(t, task)

	userID := createTestUser(t, repo, ctx)
	cmd := models.CreateExpressionCmd{
		Expression: "(5+3)*2",
		Tasks: []models.CreateExpressionCmdTask{
			{
				ID:            "task1",
				ParentTask1ID: "",
				ParentTask2ID: "",
				Arg1:          5,
				Arg2:          3,
				Operation:     models.TaskOperationAddition,
				OperationTime: time.Millisecond * 100,
			},
			{
				ID:            "task2",
				ParentTask1ID: "task1",
				ParentTask2ID: "",
				Arg1:          0,
				Arg2:          2,
				Operation:     models.TaskOperationMultiplication,
				OperationTime: time.Millisecond * 50,
			},
		},
	}

	exprID, err := repo.CreateExpression(ctx, userID, cmd)
	require.NoError(t, err, "Failed to create test expression")

	// Test getting the pending task
	task, err = repo.GetPendingTask(ctx)
	require.NoError(t, err)
	require.NotNil(t, task)

	// Verify task details
	assert.Equal(t, "task1", task.ID)
	assert.Equal(t, exprID, task.ExpressionID)
	assert.Equal(t, models.TaskStatusInProgress, task.Status)
	assert.Equal(t, models.TaskOperationAddition, task.Operation)
	assert.Equal(t, float64(5), task.Arg1.V)
	assert.Equal(t, float64(3), task.Arg2.V)

	// Try getting another pending task - should return error as there are no more pending tasks
	task, err = repo.GetPendingTask(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, models.ErrNoPendingTasks)
	require.Nil(t, task)
}

func TestRepository_FinishTask(t *testing.T) {
	db := setupTestDB(t)
	repo := New(db)
	ctx := context.Background()

	userID := createTestUser(t, repo, ctx)
	cmd := models.CreateExpressionCmd{
		Expression: "(5+3)*2",
		Tasks: []models.CreateExpressionCmdTask{
			{
				ID:            "task1",
				ParentTask1ID: "",
				ParentTask2ID: "",
				Arg1:          5,
				Arg2:          3,
				Operation:     models.TaskOperationAddition,
				OperationTime: time.Millisecond * 100,
			},
			{
				ID:            "task2",
				ParentTask1ID: "task1",
				ParentTask2ID: "",
				Arg1:          0,
				Arg2:          2,
				Operation:     models.TaskOperationMultiplication,
				OperationTime: time.Millisecond * 50,
			},
		},
	}

	_, err := repo.CreateExpression(ctx, userID, cmd)
	require.NoError(t, err, "Failed to create test expression")

	task, err := repo.GetPendingTask(ctx)
	require.NoError(t, err)
	require.NotNil(t, task)

	tests := []struct {
		name    string
		cmd     models.FinishTaskCmd
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "non-existent task",
			cmd: models.FinishTaskCmd{
				ID:     "nonexistent-task",
				Status: models.TaskStatusCompleted,
				Result: 8,
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.ErrorIs(t, err, models.ErrTaskNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.FinishTask(ctx, tt.cmd)
			tt.wantErr(t, err)
		})
	}
}

func TestRepository_FinishTask_Failed(t *testing.T) {
	db := setupTestDB(t)
	repo := New(db)
	ctx := context.Background()

	userID := createTestUser(t, repo, ctx)
	cmd := models.CreateExpressionCmd{
		Expression: "(5+3)*2",
		Tasks: []models.CreateExpressionCmdTask{
			{
				ID:            "task1",
				ParentTask1ID: "",
				ParentTask2ID: "",
				Arg1:          5,
				Arg2:          3,
				Operation:     models.TaskOperationAddition,
				OperationTime: time.Millisecond * 100,
			},
			{
				ID:            "task2",
				ParentTask1ID: "task1",
				ParentTask2ID: "",
				Arg1:          0,
				Arg2:          2,
				Operation:     models.TaskOperationMultiplication,
				OperationTime: time.Millisecond * 50,
			},
		},
	}

	exprID, err := repo.CreateExpression(ctx, userID, cmd)
	require.NoError(t, err, "Failed to create test expression")

	task, err := repo.GetPendingTask(ctx)
	require.NoError(t, err)

	failCmd := models.FinishTaskCmd{
		ID:     task.ID,
		Status: models.TaskStatusFailed,
		Result: 0,
	}
	err = repo.FinishTask(ctx, failCmd)
	require.NoError(t, err)

	// Check that the expression is marked as failed
	expr, err := repo.GetExpression(ctx, userID, exprID)
	require.NoError(t, err)
	assert.Equal(t, models.ExpressionStatusFailed, expr.Status)

	// Check that all tasks are marked as failed
	tasks, err := repo.ListExpressionTasks(ctx, userID, exprID)
	require.NoError(t, err)
	for _, task := range tasks {
		assert.Equal(t, models.TaskStatusFailed, task.Status)
	}
}

// Helper functions

func createTestUser(t *testing.T, repo *Repository, ctx context.Context) string {
	t.Helper()

	cmd := models.RegisterUserCmd{
		Login:        "testuser" + strconv.Itoa(int(time.Now().UnixNano())),
		PasswordHash: "testhash",
	}

	err := repo.Register(ctx, cmd)
	require.NoError(t, err, "Failed to create test user")

	user, err := repo.GetUser(ctx, models.GetUserCmd(cmd))
	require.NoError(t, err, "Failed to get created test user")

	return user.ID
}

func createTestExpressions(t *testing.T, repo *Repository, ctx context.Context, userID string, count int) []string {
	t.Helper()

	var exprIDs []string

	for i := 0; i < count; i++ {
		cmd := models.CreateExpressionCmd{
			Expression: "5+3",
			Tasks: []models.CreateExpressionCmdTask{
				{
					ID:            "task" + strconv.Itoa(int(time.Now().UnixNano())),
					ParentTask1ID: "",
					ParentTask2ID: "",
					Arg1:          5,
					Arg2:          3,
					Operation:     models.TaskOperationAddition,
					OperationTime: time.Millisecond * 100,
				},
			},
		}

		exprID, err := repo.CreateExpression(ctx, userID, cmd)
		require.NoError(t, err, "Failed to create test expression")

		exprIDs = append(exprIDs, exprID)

		// Small sleep to ensure different timestamps
		time.Sleep(2 * time.Millisecond)
	}

	return exprIDs
}
