package models

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrExpressionNotFound = errors.New("expression not found")
	ErrTaskNotFound       = errors.New("task not found")
	ErrNoPendingTasks     = errors.New("no pending tasks")
)

type Expression struct {
	ID         string            `db:"id"`
	UserID     string            `db:"user_id"`
	Expression string            `db:"expression"`
	Status     ExpressionStatus  `db:"status"`
	Result     sql.Null[float64] `db:"result"`
	Error      sql.Null[string]  `db:"error"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ExpressionStatus string

const (
	ExpressionStatusPending    ExpressionStatus = "Pending"
	ExpressionStatusInProgress ExpressionStatus = "InProgress"
	ExpressionStatusCompleted  ExpressionStatus = "Completed"
	ExpressionStatusFailed     ExpressionStatus = "Failed"
)

type Task struct {
	ID            string           `db:"id"`
	ExpressionID  string           `db:"expression_id"`
	ParentTask1ID sql.Null[string] `db:"parent_task_1_id"`
	ParentTask2ID sql.Null[string] `db:"parent_task_2_id"`

	Arg1          float64             `db:"arg_1"`
	Arg2          float64             `db:"arg_2"`
	Operation     TaskOperation       `db:"operation"`
	OperationTime time.Duration       `db:"operation_time"`
	Status        TaskStatus          `db:"status"`
	Result        sql.Null[float64]   `db:"result"`
	ExpireAt      sql.Null[time.Time] `db:"expire_at"` // TODO: to think

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type TaskOperation string

const (
	TaskOperationAddition       TaskOperation = "+"
	TaskOperationSubtraction    TaskOperation = "-"
	TaskOperationMultiplication TaskOperation = "*"
	TaskOperationDivision       TaskOperation = "/"
)

type TaskStatus string

const (
	TaskStatusCreated    TaskStatus = "Created"
	TaskStatusPending    TaskStatus = "Pending"
	TaskStatusInProgress TaskStatus = "InProgress"
	TaskStatusCompleted  TaskStatus = "Completed"
	TaskStatusFailed     TaskStatus = "Failed"
)

type CreateExpressionCmd struct {
	Expression string
	Tasks      []CreateExpressionCmdTask
}

type CreateExpressionCmdTask struct {
	ID            string
	ParentTask1ID string
	ParentTask2ID string

	Arg1          float64
	Arg2          float64
	Operation     TaskOperation
	OperationTime time.Duration
}

type FinishTaskCmd struct {
	ID     string
	Status TaskStatus
	Result float64
}
