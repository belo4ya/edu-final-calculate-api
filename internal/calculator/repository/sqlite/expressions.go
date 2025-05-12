package sqlite

import (
	"context"
	"database/sql"
	"edu-final-calculate-api/internal/calculator/repository/sqlite/models"
	"errors"
	"fmt"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/rs/xid"
)

// CreateExpression stores a new expression with its associated tasks
// and returns the ID of the created expression.
func (r *Repository) CreateExpression(ctx context.Context, userID string, cmd models.CreateExpressionCmd) (string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const q = `
        INSERT INTO expressions (id, user_id, expression, status, created_at, updated_at)
		VALUES (:id, :user_id, :expression, :status, :created_at, :updated_at)
    `

	now := time.Now().UTC()
	expr := models.Expression{
		ID:         xid.New().String(),
		UserID:     userID,
		Expression: cmd.Expression,
		Status:     models.ExpressionStatusPending,
		Result:     sql.Null[float64]{},
		Error:      sql.Null[string]{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if _, err = tx.NamedExecContext(ctx, q, expr); err != nil {
		return "", fmt.Errorf("db exec: %w", err)
	}

	tasks := make([]models.Task, 0, len(cmd.Tasks))
	for _, t := range cmd.Tasks {
		status := models.TaskStatusCreated
		if t.ParentTask1ID == "" && t.ParentTask2ID == "" {
			status = models.TaskStatusPending
		}
		tasks = append(tasks, models.Task{
			ID:            t.ID,
			ExpressionID:  expr.ID,
			ParentTask1ID: sql.Null[string]{V: t.ParentTask1ID, Valid: t.ParentTask1ID != ""},
			ParentTask2ID: sql.Null[string]{V: t.ParentTask2ID, Valid: t.ParentTask2ID != ""},
			Arg1:          t.Arg1,
			Arg2:          t.Arg2,
			Operation:     t.Operation,
			OperationTime: t.OperationTime,
			Status:        status,
			Result:        sql.Null[float64]{},
			ExpireAt:      sql.Null[time.Time]{},
			CreatedAt:     expr.CreatedAt,
			UpdatedAt:     expr.UpdatedAt,
		})
	}

	sb := sqlbuilder.NewInsertBuilder().InsertInto("tasks").Cols(
		"id",
		"expression_id",
		"parent_task_1_id",
		"parent_task_2_id",
		"arg1",
		"arg2",
		"operation",
		"operation_time",
		"status",
		"created_at",
		"updated_at",
	)
	for _, t := range tasks {
		sb.Values(
			t.ID,
			t.ExpressionID,
			t.ParentTask1ID,
			t.ParentTask2ID,
			t.Arg1,
			t.Arg2,
			t.Operation,
			t.OperationTime,
			t.Status,
			t.CreatedAt,
			t.UpdatedAt,
		)
	}

	query, args := sb.Build()
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return "", fmt.Errorf("db query: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	return expr.ID, nil
}

// ListExpressions retrieves all stored expressions for a specific user.
func (r *Repository) ListExpressions(ctx context.Context, userID string) ([]models.Expression, error) {
	const q = `
        SELECT id, user_id, expression, status, result, error, created_at, updated_at 
        FROM expressions 
        WHERE user_id = ?
        ORDER BY created_at DESC
    `

	var exprs []models.Expression
	if err := r.db.SelectContext(ctx, &exprs, q, userID); err != nil {
		return nil, fmt.Errorf("db select: %w", err)
	}

	return exprs, nil
}

// GetExpression retrieves a specific expression by its ID for a specific user.
// Returns [models.ErrExpressionNotFound] if the expression doesn't exist.
func (r *Repository) GetExpression(ctx context.Context, userID string, exprID string) (*models.Expression, error) {
	const q = `
        SELECT id, user_id, expression, status, result, error, created_at, updated_at 
        FROM expressions 
        WHERE id = ? AND user_id = ?
    `

	var expr models.Expression
	if err := r.db.GetContext(ctx, &expr, q, exprID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrExpressionNotFound
		}
		return nil, fmt.Errorf("db get: %w", err)
	}

	return &expr, nil
}

// ListExpressionTasks retrieves all tasks associated with a specific expression for a specific user.
// Returns [models.ErrExpressionNotFound] if the expression doesn't exist.
func (r *Repository) ListExpressionTasks(ctx context.Context, userID string, exprID string) ([]models.Task, error) {
	q := `SELECT COUNT(*) FROM expressions WHERE id = ? AND user_id = ?`

	var count int
	if err := r.db.GetContext(ctx, &count, q, exprID, userID); err != nil {
		return nil, fmt.Errorf("db get: %w", err)
	}
	if count == 0 {
		return nil, models.ErrExpressionNotFound
	}

	q = `
        SELECT id, expression_id, parent_task_1_id, parent_task_2_id, 
               arg1, arg2, operation, operation_time, status, result, expire_at,
               created_at, updated_at
        FROM tasks 
        WHERE expression_id = ?
        ORDER BY created_at
    `

	var tasks []models.Task
	if err := r.db.SelectContext(ctx, &tasks, q, exprID); err != nil {
		return nil, fmt.Errorf("db select: %w", err)
	}

	return tasks, nil
}

// GetPendingTask retrieves and claims the first available pending task.
// Returns [models.ErrNoPendingTasks] if there are no pending tasks available.
func (r *Repository) GetPendingTask(ctx context.Context) (models.Task, error) {
	//TODO implement me
	panic("implement me")
}

// FinishTask updates a task's status and result, and handles subsequent operations
// like updating related tasks, enqueueing child tasks, or completing expressions.
// Returns [models.ErrTaskNotFound] if the task doesn't exist.
func (r *Repository) FinishTask(ctx context.Context, cmd models.FinishTaskCmd) error {
	//TODO implement me
	panic("implement me")
}
