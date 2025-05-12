package service

import (
	"edu-final-calculate-api/internal/calculator/repository/sqlite/models"

	calculatorv1 "edu-final-calculate-api/pkg/calculator/v1"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapExpressionToExpressionResponse(expr *models.Expression) *calculatorv1.Expression {
	return &calculatorv1.Expression{
		Id:         expr.ID,
		Expression: expr.Expression,
		Status:     mapExpressionStatus(expr.Status),
		Result:     expr.Result.V,
	}
}

func mapTaskToAgentTaskResponse(task *models.Task) *calculatorv1.Task {
	return &calculatorv1.Task{
		Id:            task.ID,
		Arg1:          task.Arg1,
		Arg2:          task.Arg2,
		Operation:     mapTaskOperation(task.Operation),
		OperationTime: durationpb.New(task.OperationTime),
	}
}

func mapTaskToInternalTaskResponse(task models.Task) *calculatorv1.ListExpressionTasksResponse_Task {
	return &calculatorv1.ListExpressionTasksResponse_Task{
		Id:             task.ID,
		ExpressionId:   task.ExpressionID,
		ParentTask_1Id: task.ParentTask1ID.V,
		ParentTask_2Id: task.ParentTask2ID.V,
		Arg_1:          task.Arg1,
		Arg_2:          task.Arg2,
		Operation:      mapTaskOperation(task.Operation),
		OperationTime:  durationpb.New(task.OperationTime),
		Status:         mapTaskStatus(task.Status),
		Result:         task.Result.V,
		ExpireAt:       timestamppb.New(task.ExpireAt.V),
		CreatedAt:      timestamppb.New(task.CreatedAt),
		UpdatedAt:      timestamppb.New(task.UpdatedAt),
	}
}

func mapExpressionStatus(s models.ExpressionStatus) calculatorv1.ExpressionStatus {
	switch s {
	case models.ExpressionStatusPending:
		return calculatorv1.ExpressionStatus_EXPRESSION_STATUS_PENDING
	case models.ExpressionStatusInProgress:
		return calculatorv1.ExpressionStatus_EXPRESSION_STATUS_IN_PROGRESS
	case models.ExpressionStatusCompleted:
		return calculatorv1.ExpressionStatus_EXPRESSION_STATUS_COMPLETED
	case models.ExpressionStatusFailed:
		return calculatorv1.ExpressionStatus_EXPRESSION_STATUS_FAILED
	default:
		return calculatorv1.ExpressionStatus_EXPRESSION_STATUS_UNSPECIFIED
	}
}

func mapTaskOperation(s models.TaskOperation) calculatorv1.TaskOperation {
	switch s {
	case models.TaskOperationAddition:
		return calculatorv1.TaskOperation_TASK_OPERATION_ADDITION
	case models.TaskOperationSubtraction:
		return calculatorv1.TaskOperation_TASK_OPERATION_SUBTRACTION
	case models.TaskOperationMultiplication:
		return calculatorv1.TaskOperation_TASK_OPERATION_MULTIPLICATION
	case models.TaskOperationDivision:
		return calculatorv1.TaskOperation_TASK_OPERATION_DIVISION
	default:
		return calculatorv1.TaskOperation_TASK_OPERATION_UNSPECIFIED
	}
}

func mapTaskStatus(s models.TaskStatus) calculatorv1.TaskStatus {
	switch s {
	case models.TaskStatusCreated:
		return calculatorv1.TaskStatus_TASK_STATUS_CREATED
	case models.TaskStatusPending:
		return calculatorv1.TaskStatus_TASK_STATUS_PENDING
	case models.TaskStatusInProgress:
		return calculatorv1.TaskStatus_TASK_STATUS_IN_PROGRESS
	case models.TaskStatusCompleted:
		return calculatorv1.TaskStatus_TASK_STATUS_COMPLETED
	case models.TaskStatusFailed:
		return calculatorv1.TaskStatus_TASK_STATUS_FAILED
	default:
		return calculatorv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}
