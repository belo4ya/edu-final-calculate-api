package sqlite

import (
	"context"
	"database/sql"
	"edu-final-calculate-api/internal/calculator/repository/sqlite/models"
	"errors"
	"fmt"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jmoiron/sqlx"
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

	sb := sqlbuilder.InsertInto("tasks").Cols(
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
func (r *Repository) GetPendingTask(ctx context.Context) (*models.Task, error) {
	const q = `
        UPDATE tasks
		SET status     = :status_in_progress,
			updated_at = :updated_at
		WHERE id = ( SELECT id FROM tasks WHERE status = :status_pending ORDER BY created_at LIMIT 1 )
		RETURNING id, expression_id, parent_task_1_id, parent_task_2_id,
			arg1, arg2, operation, operation_time, status, result,
			expire_at, created_at, updated_at
    `

	row, err := r.db.NamedQueryContext(ctx, q, map[string]any{
		"status_in_progress": models.TaskStatusInProgress,
		"updated_at":         time.Now().UTC(),
		"status_pending":     models.TaskStatusPending,
	})
	if err != nil {
		return nil, fmt.Errorf("db query: %w", err)
	}
	defer func(row *sqlx.Rows) {
		_ = row.Close()
	}(row)

	if !row.Next() {
		return nil, models.ErrNoPendingTasks
	}

	var task models.Task
	if err := row.StructScan(&task); err != nil {
		return nil, fmt.Errorf("scan task: %w", err)
	}

	return &task, nil
}

// FinishTask updates a task's status and result, and handles subsequent operations
// like updating related tasks, enqueueing child tasks, or completing expressions.
// Returns [models.ErrTaskNotFound] if the task doesn't exist.
func (r *Repository) FinishTask(ctx context.Context, cmd models.FinishTaskCmd) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now()

	// Update the task with the result
	result, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET status = ?, 
		    result = ?, 
		    updated_at = ?,
		    expire_at = NULL
		WHERE id = ?
	`, cmd.Status, cmd.Result, now, cmd.TaskID)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", cmd.TaskID)
	}

	// Check if we need to update the expression status
	var expressionID string
	err = tx.QueryRowContext(ctx, "SELECT expression_id FROM tasks WHERE id = ?", cmd.TaskID).Scan(&expressionID)
	if err != nil {
		return fmt.Errorf("failed to get expression ID: %w", err)
	}

	// Count pending/in-progress tasks for this expression
	var unfinishedCount int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM tasks 
		WHERE expression_id = ? AND status IN ('pending', 'in_progress')
	`, expressionID).Scan(&unfinishedCount)

	if err != nil {
		return fmt.Errorf("failed to count unfinished tasks: %w", err)
	}

	// If no tasks are pending, the expression is completed
	if unfinishedCount == 0 {
		// Find the root task (which has no parents) to get the final result
		var rootResult float64
		err = tx.QueryRowContext(ctx, `
			SELECT result 
			FROM tasks 
			WHERE expression_id = ? AND parent_task_1_id IS NULL AND parent_task_2_id IS NULL
		`, expressionID).Scan(&rootResult)

		if err != nil {
			return fmt.Errorf("failed to get root task result: %w", err)
		}

		// Update the expression with the final result
		_, err = tx.ExecContext(ctx, `
			UPDATE expressions
			SET status = 'completed', 
			    result = ?,
			    updated_at = ?
			WHERE id = ?
		`, rootResult, now, expressionID)

		if err != nil {
			return fmt.Errorf("failed to update expression: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
