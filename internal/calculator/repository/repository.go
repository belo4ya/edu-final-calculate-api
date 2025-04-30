package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"edu-final-calculate-api/internal/calculator/repository/models"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/xid"
)

// Repository provides storage operations for calculator expressions and tasks.
type Repository struct {
	db *badger.DB
}

// New creates a new repository instance with the provided BadgerDB.
func New(db *badger.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Register(ctx context.Context, cmd models.RegisterUserCmd) error {
	//TODO implement me
	panic("implement me")
}

func (r *Repository) GetUser(ctx context.Context, cmd models.GetUserCmd) (models.User, error) {
	//TODO implement me
	panic("implement me")
}

// CreateExpression stores a new expression with its associated tasks
// and returns the ID of the created expression.
func (r *Repository) CreateExpression(
	_ context.Context,
	exprCmd models.CreateExpressionCmd,
	tasksCmd []models.CreateExpressionTaskCmd,
) (string, error) {
	timeNow := time.Now().UTC()

	expr := models.Expression{
		ID:         xid.New().String(),
		Expression: exprCmd.Expression,
		Status:     models.ExpressionStatusPending,
		CreatedAt:  timeNow,
		UpdatedAt:  timeNow,
	}

	tasks := make([]models.Task, 0, len(tasksCmd))
	for _, t := range tasksCmd {
		tasks = append(tasks, models.Task{
			ID:            t.ID,
			ExpressionID:  expr.ID,
			ParentTask1ID: t.ParentTask1ID,
			ParentTask2ID: t.ParentTask2ID,
			Arg1:          t.Arg1,
			Arg2:          t.Arg2,
			Operation:     t.Operation,
			OperationTime: t.OperationTime,
			Status:        models.TaskStatusPending,
			CreatedAt:     timeNow,
			UpdatedAt:     timeNow,
		})
	}

	taskToChildTask := map[string]string{}
	for _, task := range tasks {
		if task.ParentTask1ID != "" {
			taskToChildTask[task.ParentTask1ID] = task.ID
		}
		if task.ParentTask2ID != "" {
			taskToChildTask[task.ParentTask2ID] = task.ID
		}
	}

	err := r.db.Update(func(txn *badger.Txn) error {
		if err := setVal(txn, exprKey(expr.ID), expr); err != nil {
			return fmt.Errorf("store expr: %w", err)
		}
		if err := setVal(txn, exprListKey(expr.ID), expr.ID); err != nil {
			return fmt.Errorf("add to expr list: %w", err)
		}

		for _, task := range tasks {
			if err := setVal(txn, taskKey(task.ID), task); err != nil {
				return fmt.Errorf("store task: %w", err)
			}
			if err := setOnlyKey(txn, exprTaskKey(expr.ID, task.ID)); err != nil {
				return fmt.Errorf("add to expr's task list: %w", err)
			}

			// Set up task relationships - either mark as child task or final expression task
			if childTaskID, ok := taskToChildTask[task.ID]; ok {
				if err := setOnlyKey(txn, taskChildKey(task.ID, childTaskID)); err != nil {
					return fmt.Errorf("set task's child task: %w", err)
				}
			} else {
				if err := setOnlyKey(txn, exprFinalTaskKey(expr.ID, task.ID)); err != nil {
					return fmt.Errorf("set expr's final task: %w", err)
				}
			}

			// Add root tasks (no parents) to the pending queue for immediate processing
			if task.ParentTask1ID == "" && task.ParentTask2ID == "" {
				if err := setOnlyKey(txn, taskQueuePendingKey(task.ID)); err != nil {
					return fmt.Errorf("enque task: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}
	return expr.ID, nil
}

// ListExpressions retrieves all stored expressions.
func (r *Repository) ListExpressions(_ context.Context) ([]models.Expression, error) {
	var exprs []models.Expression

	err := r.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := exprListPrefix()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			exprID := exprIDFromListKey(it.Item().Key())

			var expr models.Expression
			if err := scanVal(txn, exprKey(exprID), &expr); err != nil {
				return fmt.Errorf("get expr: %w", err)
			}
			exprs = append(exprs, expr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return exprs, nil
}

// GetExpression retrieves a specific expression by its ID.
// Returns models.ErrExpressionNotFound if the expression doesn't exist.
func (r *Repository) GetExpression(_ context.Context, id string) (models.Expression, error) {
	var expr models.Expression

	err := r.db.View(func(txn *badger.Txn) error {
		if err := scanVal(txn, exprKey(id), &expr); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return models.ErrExpressionNotFound
			}
			return fmt.Errorf("get expr: %w", err)
		}
		return nil
	})

	if err != nil {
		return models.Expression{}, err
	}
	return expr, nil
}

// GetPendingTask retrieves and claims the first available pending task.
// Returns models.ErrNoPendingTasks if there are no pending tasks available.
func (r *Repository) GetPendingTask(_ context.Context) (models.Task, error) {
	var task models.Task

	err := r.db.Update(func(txn *badger.Txn) error {
		// Find and retrieve first pending task from queue
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := taskQueuePendingPrefix()
		it.Seek(prefix)

		if !it.ValidForPrefix(prefix) {
			return models.ErrNoPendingTasks
		}

		taskID := taskIDFromPendingQueueKey(it.Item().Key())

		if err := txn.Delete(it.Item().Key()); err != nil {
			return fmt.Errorf("delete task from queue: %q", err)
		}

		if err := scanVal(txn, taskKey(taskID), &task); err != nil {
			return fmt.Errorf("get task: %w", err)
		}

		// Update task state to in-progress
		timeNow := time.Now().UTC()
		task.Status = models.TaskStatusInProgress
		task.UpdatedAt = timeNow
		task.ExpireAt = timeNow.Add(2 * (task.OperationTime + time.Minute)) // TODO: to think
		if err := setVal(txn, taskKey(taskID), task); err != nil {
			return fmt.Errorf("to in-progress task: %w", err)
		}

		// Update parent expression state if this is the first task being processed
		var expr models.Expression
		if err := scanVal(txn, exprKey(task.ExpressionID), &expr); err != nil {
			return fmt.Errorf("get expr: %w", err)
		}

		if expr.Status == models.ExpressionStatusPending {
			expr.Status = models.ExpressionStatusInProgress
			expr.UpdatedAt = timeNow
			if err := setVal(txn, exprKey(expr.ID), expr); err != nil {
				return fmt.Errorf("to in-progress expr: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return models.Task{}, err
	}
	return task, nil
}

// FinishTask updates a task's status and result, and handles subsequent operations
// like updating related tasks, enqueueing child tasks, or completing expressions.
// Returns models.ErrTaskNotFound if the task doesn't exist.
func (r *Repository) FinishTask(_ context.Context, cmd models.FinishTaskCmd) error {
	return r.db.Update(func(txn *badger.Txn) error {
		if cmd.Status != models.TaskStatusCompleted && cmd.Status != models.TaskStatusFailed {
			return fmt.Errorf("unexpected task status: %s", cmd.Status)
		}

		// Retrieve and update the task
		var task models.Task
		if err := scanVal(txn, taskKey(cmd.ID), &task); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return models.ErrTaskNotFound
			}
			return fmt.Errorf("get task: %w", err)
		}

		task.Status = cmd.Status
		task.Result = cmd.Result
		task.UpdatedAt = time.Now().UTC()
		if err := setVal(txn, taskKey(task.ID), task); err != nil {
			return fmt.Errorf("update task: %w", err)
		}

		// Handle task failure - propagate failure to entire expression
		if task.Status == models.TaskStatusFailed {
			if err := r.failExpression(txn, task.ExpressionID); err != nil {
				return fmt.Errorf("fail expr: %w", err)
			}
			return nil
		}

		// Process successfully completed task - either enqueue child or complete expression
		isFinal, err := r.isFinalTask(txn, task)
		if err != nil {
			return fmt.Errorf("is final task: %w", err)
		}

		if !isFinal {
			if err := r.enqueueChildTask(txn, task); err != nil {
				return fmt.Errorf("enqueue child task: %w", err)
			}
		} else {
			if err := r.completeExpression(txn, task.ExpressionID, task); err != nil {
				return fmt.Errorf("complete expr: %w", err)
			}
		}

		return nil
	})
}

// ListExpressionTasks retrieves all tasks associated with a specific expression.
// Returns models.ErrExpressionNotFound if the expression doesn't exist.
func (r *Repository) ListExpressionTasks(_ context.Context, id string) ([]models.Task, error) {
	var tasks []models.Task

	err := r.db.View(func(txn *badger.Txn) error {
		// First verify expression exists
		if _, err := txn.Get(exprKey(id)); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return models.ErrExpressionNotFound
			}
			return fmt.Errorf("get %q:%w", string(exprKey(id)), err)
		}

		// Collect all tasks for this expression
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := exprTasksPrefix(id)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			taskID := taskIDFromExprTaskKey(it.Item().Key(), id)

			var task models.Task
			if err := scanVal(txn, taskKey(taskID), &task); err != nil {
				return fmt.Errorf("get task: %w", err)
			}
			tasks = append(tasks, task)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *Repository) isFinalTask(txn *badger.Txn, task models.Task) (bool, error) {
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()

	it.Seek(exprFinalTaskPrefix(task.ExpressionID))

	finalTaskID := taskIDFromExprFinalTaskKey(it.Item().Key(), task.ExpressionID)
	return task.ID == finalTaskID, nil
}

func (r *Repository) enqueueChildTask(txn *badger.Txn, completedTask models.Task) error {
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()

	// Find child task that depends on the completed task
	it.Seek(taskChildPrefix(completedTask.ID))
	if !it.ValidForPrefix(taskChildPrefix(completedTask.ID)) {
		return nil // should not happen
	}

	childTaskID := taskIDFromTaskChildKey(it.Item().Key(), completedTask.ID)

	var childTask models.Task
	if err := scanVal(txn, taskKey(childTaskID), &childTask); err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	// Update child task with parent's result value
	if childTask.ParentTask1ID == completedTask.ID {
		childTask.Arg1 = completedTask.Result
	} else { // childTask.ParentTask2ID == completedTask.ID
		childTask.Arg2 = completedTask.Result
	}
	childTask.UpdatedAt = time.Now().UTC()

	if err := setVal(txn, taskKey(childTask.ID), childTask); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	// Check if both parents are complete and the task is ready to be queued
	var parent1 models.Task
	if childTask.ParentTask1ID != "" {
		if err := scanVal(txn, taskKey(childTask.ParentTask1ID), &parent1); err != nil {
			return fmt.Errorf("get parent 1 of task: %w", err) // 👨‍👩‍👦 😅
		}
	}
	var parent2 models.Task
	if childTask.ParentTask2ID != "" {
		if err := scanVal(txn, taskKey(childTask.ParentTask2ID), &parent2); err != nil {
			return fmt.Errorf("get parent 2 of task: %w", err) // 👨‍👩‍👦 😅
		}
	}

	if (childTask.ParentTask1ID == "" || parent1.Status == models.TaskStatusCompleted) &&
		(childTask.ParentTask2ID == "" || parent2.Status == models.TaskStatusCompleted) {
		if err := setOnlyKey(txn, taskQueuePendingKey(childTask.ID)); err != nil {
			return fmt.Errorf("enqueue task: %w", err)
		}
	}
	return nil
}

func (r *Repository) failExpression(txn *badger.Txn, exprID string) error {
	// Mark all unfinished tasks as failed
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()

	prefix := exprTasksPrefix(exprID)
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		taskID := taskIDFromExprTaskKey(it.Item().Key(), exprID)

		var task models.Task
		if err := scanVal(txn, taskKey(taskID), &task); err != nil {
			return fmt.Errorf("get task: %w", err)
		}

		if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusFailed {
			continue
		}

		_ = txn.Delete(taskQueuePendingKey(task.ID))

		task.Status = models.TaskStatusFailed
		task.UpdatedAt = time.Now().UTC()
		if err := setVal(txn, taskKey(task.ID), task); err != nil {
			return fmt.Errorf("update task: %w", err)
		}
	}

	// Mark the expression as failed
	var expr models.Expression
	if err := scanVal(txn, exprKey(exprID), &expr); err != nil {
		return fmt.Errorf("get expr: %w", err)
	}

	expr.Status = models.ExpressionStatusFailed
	expr.UpdatedAt = time.Now().UTC()
	if err := setVal(txn, exprKey(exprID), expr); err != nil {
		return fmt.Errorf("update expr: %w", err)
	}

	return nil
}

func (r *Repository) completeExpression(txn *badger.Txn, exprID string, finalTask models.Task) error {
	var expr models.Expression
	if err := scanVal(txn, exprKey(exprID), &expr); err != nil {
		return fmt.Errorf("get expr: %w", err)
	}

	expr.Status = models.ExpressionStatusCompleted
	expr.Result = finalTask.Result
	expr.UpdatedAt = time.Now().UTC()
	if err := setVal(txn, exprKey(expr.ID), expr); err != nil {
		return fmt.Errorf("update expr: %w", err)
	}

	return nil
}
