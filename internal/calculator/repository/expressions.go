package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/repository/models"

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
			Arg1:          sql.Null[float64]{V: t.Arg1, Valid: t.ParentTask1ID == ""},
			Arg2:          sql.Null[float64]{V: t.Arg2, Valid: t.ParentTask2ID == ""},
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
		"id", "expression_id", "parent_task_1_id", "parent_task_2_id",
		"arg1", "arg2", "operation", "operation_time", "status", "created_at", "updated_at",
	)
	for _, t := range tasks {
		sb.Values(
			t.ID, t.ExpressionID, t.ParentTask1ID, t.ParentTask2ID,
			t.Arg1, t.Arg2, t.Operation, t.OperationTime, t.Status, t.CreatedAt, t.UpdatedAt,
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
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const q = `
        UPDATE tasks
        SET status = :status,
            result = :result,
            updated_at = :updated_at
        WHERE id = :id
        RETURNING id, expression_id, parent_task_1_id, parent_task_2_id,
			arg1, arg2, operation, operation_time, status, result,
			expire_at, created_at, updated_at
    `

	row, err := sqlx.NamedQueryContext(ctx, tx, q, map[string]any{
		"status":     cmd.Status,
		"result":     sql.Null[float64]{V: cmd.Result, Valid: cmd.Status == models.TaskStatusCompleted},
		"updated_at": time.Now().UTC(),
		"id":         cmd.ID,
	})
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	defer func(row *sqlx.Rows) {
		_ = row.Close()
	}(row)

	if !row.Next() {
		return models.ErrTaskNotFound
	}

	var task models.Task
	if err := row.StructScan(&task); err != nil {
		return fmt.Errorf("scan task: %w", err)
	}

	// Handle task failure - propagate failure to entire expression
	if cmd.Status == models.TaskStatusFailed {
		if err := r.failExpression(ctx, tx, &task); err != nil {
			return fmt.Errorf("fail expr: %w", err)
		}
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("commit transaction: %w", err)
		}
		return nil
	}

	// Process successfully completed task - either enqueue child or complete expression
	isFinal, err := r.isFinalTask(ctx, tx, task.ID)
	if err != nil {
		return fmt.Errorf("is final task: %w", err)
	}

	if !isFinal {
		if err := r.enqueueChildTask(ctx, tx, &task); err != nil {
			return fmt.Errorf("enqueue child task: %w", err)
		}
	} else {
		if err := r.completeExpression(ctx, tx, task.ExpressionID, &task); err != nil {
			return fmt.Errorf("complete expr: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (r *Repository) failExpression(ctx context.Context, tx *sqlx.Tx, task *models.Task) error {
	q := `UPDATE expressions SET status = ?, updated_at = ? WHERE id = ?`
	if _, err := tx.ExecContext(
		ctx, q,
		models.ExpressionStatusFailed, task.UpdatedAt, task.ExpressionID,
	); err != nil {
		return fmt.Errorf("fail expression: %w", err)
	}

	q = `UPDATE tasks SET status = ?, updated_at = ? WHERE expression_id = ? AND status NOT IN (?, ?)`
	if _, err := tx.ExecContext(
		ctx, q,
		models.TaskStatusFailed,
		task.UpdatedAt,
		task.ExpressionID,
		models.TaskStatusCompleted,
		models.TaskStatusFailed,
	); err != nil {
		return fmt.Errorf("tx exec: %w", err)
	}
	return nil
}

func (r *Repository) isFinalTask(ctx context.Context, tx *sqlx.Tx, taskID string) (bool, error) {
	const q = `SELECT COUNT(*) FROM tasks WHERE parent_task_1_id = ? OR parent_task_2_id = ?`

	var child int
	if err := tx.GetContext(ctx, &child, q, taskID, taskID); err != nil {
		return false, fmt.Errorf("check child tasks: %w", err)
	}
	return child == 0, nil
}

func (r *Repository) enqueueChildTask(ctx context.Context, tx *sqlx.Tx, completedTask *models.Task) error {
	// Find child task that depends on the completed task
	q := `
			SELECT id,
				   expression_id,
				   parent_task_1_id,
				   parent_task_2_id,
				   arg1,
				   arg2,
				   operation,
				   operation_time,
				   status,
				   result,
				   expire_at,
				   created_at,
				   updated_at
			FROM tasks
			WHERE parent_task_1_id = :task_id
			   OR parent_task_2_id = :task_id
		`

	row, err := sqlx.NamedQueryContext(ctx, tx, q, map[string]any{"task_id": completedTask.ID})
	if err != nil {
		return fmt.Errorf("select child task: %w", err)
	}
	defer func(row *sqlx.Rows) {
		_ = row.Close()
	}(row)

	if !row.Next() {
		return errors.New("child task not found")
	}

	var childTask models.Task
	if err := row.StructScan(&childTask); err != nil {
		return fmt.Errorf("scan task: %w", err)
	}

	// Update child task with parent's result value
	if childTask.ParentTask1ID.Valid && childTask.ParentTask1ID.V == completedTask.ID {
		childTask.Arg1 = completedTask.Result
	} else { // childTask.ParentTask2ID == completedTask.ID
		childTask.Arg2 = completedTask.Result
	}
	if childTask.Arg1.Valid && childTask.Arg2.Valid {
		childTask.Status = models.TaskStatusPending
	}
	childTask.UpdatedAt = completedTask.UpdatedAt

	q = `
			UPDATE tasks
			SET arg1       = :arg1,
				arg2       = :arg2,
				status     = :status,
				updated_at = :updated_at
			WHERE id = :id
		`

	if _, err := tx.NamedExecContext(ctx, q, childTask); err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func (r *Repository) completeExpression(ctx context.Context, tx *sqlx.Tx, exprID string, finalTask *models.Task) error {
	const q = `UPDATE expressions SET status = ?, result = ?, updated_at = ? WHERE id = ?`

	if _, err := tx.ExecContext(
		ctx, q,
		models.ExpressionStatusCompleted, finalTask.Result, finalTask.UpdatedAt, exprID,
	); err != nil {
		return fmt.Errorf("update expr: %w", err)
	}
	return nil
}
