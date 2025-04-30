package models

import "time"

type CreateExpressionCmd struct {
	Expression string
}

type CreateExpressionTaskCmd struct {
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

// Помоги переписать реализацию repository.Repository с kv-базы данных на SQLite. Начни с изучения моделей (и команд), на основе моделей предложи нормализованную модель данных (схему данных до 3-ей нормальной формы). При необходимости предложи изменения для моделей которые необходимы для соблюдения 3-х нормальных форм
