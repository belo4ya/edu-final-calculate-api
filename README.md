# Edu Final Calculate API

![Go Version](https://img.shields.io/github/go-mod/go-version/belo4ya/edu-final-calculate-api?logo=go)
[![GoDoc](https://godoc.org/github.com/belo4ya/edu-final-calculate-api?status.svg)](https://pkg.go.dev/github.com/belo4ya/edu-final-calculate-api)
[![Go report](https://goreportcard.com/badge/github.com/belo4ya/edu-final-calculate-api)](https://goreportcard.com/report/github.com/belo4ya/edu-final-calculate-api)

> 🚧 Ведутся незначительные доработки. Для оценки зайди, пожалуйста, завтра 🚧

> 🛠 Ведутся незначительные доработки. Для оценки зайди, пожалуйста, завтра 🛠

> 🔜 Ведутся незначительные доработки. Для оценки зайди, пожалуйста, завтра 🔜

Программирование на Go | 24. Распределенный вычислитель арифметических выражений.
Пользователь отправляет арифметическое выражение по HTTP и получает в ответ ~~его результат~~ id задачи вычисления.

Сервис состоит из двух частей:

- [cmd/calculator](cmd/calculator) - реализует API для распределенного вычисления арифметических выражений
  и выполняет роль оркестратора задач.
- [cmd/agent](cmd/agent) - вычислитель, который может получить от оркестратора задачу,
  выполнить его и вернуть серверу результат.

## 📚 Решения и особенности

HTTP API реализован с помощью [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway)
по верх grpc (см. [api/](api)).
Было лень возиться с json-ами 🙄, брать фреймворк по типу fiber/echo/gin тоже лень.
Решил поразбираться с [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway)
и [buf.build](https://buf.build/) для генерации.

Для генерации mock'ов используется [mockery](https://github.com/vektra/mockery)
(см. [internal/testutil/](internal/testutil)).

Калькулятор ([calculator/calc/](internal/calculator/calc)) - не самая сильная часть этого приложения,
можно убедиться в этом по тестам с флагом skip [calculator/calc/calc_test.go](internal/calculator/calc/calc_test.go).

## 🔧 Конфигурация

Конфигурация выполняется с помощью переменных окружения. Значения по умолчанию можно найти в
[calculator/config/config.go](internal/calculator/config/config.go)
и [agent/config/config.go](internal/agent/config/config.go).

Необходимые значения также можно задать с помощью .env-файлов `.env.calculator` и `.env.agent`
(см. примеры [.env.calculator.example](.env.calculator.example) и [.env.agent.example](.env.agent.example))

### Calculator

- `LOG_LEVEL`: Уровень логирования (по умолчанию: `info`)
- `MGMT_ADDR`: Адрес сервера управления (по умолчанию: `:8081`)
- `GRPC_ADDR`: Адрес GRPC сервера (по умолчанию: `:50051`)
- `HTTP_ADDR`: Адрес HTTP сервера (по умолчанию: `:8080`)
- `DB_SQLITE_PATH`: Путь к хранилищу базы данных SQLite (по умолчанию: `.data/db.sqlite`)
- `AUTH_JWT_SECRET`: TODO (по умолчанию: `jwt-secret`)
- `AUTH_JWT_EXPIRATION_TIME`: TODO (по умолчанию: `1h`)
- `TIME_ADDITION_MS`: Время в миллисекундах для операций сложения (по умолчанию: `1000`)
- `TIME_SUBTRACTION_MS`: Время в миллисекундах для операций вычитания (по умолчанию: `1000`)
- `TIME_MULTIPLICATION_MS`: Время в миллисекундах для операций умножения (по умолчанию: `1000`)
- `TIME_DIVISION_MS`: Время в миллисекундах для операций деления (по умолчанию: `1000`)

### Agent

- `LOG_LEVEL`: Уровень логирования (по умолчанию: `info`)
- `MGMT_ADDR`: Адрес сервера управления (по умолчанию: `:8082`)
- `CALCULATOR_API_ADDR`: Адрес сервиса Calculator API (по умолчанию: `localhost:50051`)
- `COMPUTING_POWER`: Количество одновременных вычислительных задач (по умолчанию: `4`)

## 🚀 Запуск

Запустить все компоненты сервиса:

```shell
make up
```

Для тех у кого болит Docker

```shell
make migrate
go run ./cmd/calculator &
go run ./cmd/agent &
# не забудь остановить процессы с помощью kill <pid>
```

Запуск рутины:

```shell
go mod tidy
make generate # requires buf: brew install bufbuild/buf/buf (см. https://buf.build/docs/cli/installation/)
make lint test-cov
```

## 💡 Использование

Сервис предоставляет HTTP и GRPC API. По умолчанию HTTP API доступно по
адресу [http://localhost:8080](http://localhost:8080), а GRPC - [http://localhost:50051](http://localhost:50051).

Спецификацию API можно найти в [api/calculator/v1](api/calculator/v1)
или [api/api.swagger.json](api/api.swagger.json).

Вместо UI интерактивно поработать с HTTP API сервиса можно с помощью SwaggerUI,
доступного по адресу [http://localhost:8080/docs/](http://localhost:8080/docs/).

<details>
<summary>Картинка</summary>
<img src="docs/assets/swagger-ui.png" alt="" width="800">
</details>

Для работы с [Expressions API](#expressions-api) (`/api/v1/calculate`, `/api/v1/expressions`)
необходима авторизация с помощью Access Token со схемой Bearer.
По умолчанию доступен пользователь `admin` с паролем `admin`
(см. [Примеры curl](#примеры-curl) и [User API](#users-api)):

```shell
curl -X 'POST' 'http://localhost:8080/api/v1/login' \
  -d '{
  "login": "admin",
  "password": "admin"
}'

#{
#  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2luZm8iOnsiaWQiOiIwMDAwMDAwMDAwMDAwMDAwMDAwMCIsImxvZ2luIjoiYWRtaW4ifSwic3ViIjoiMDAwMDAwMDAwMDAwMDAwMDAwMDAiLCJleHAiOjE3NDcwODM0NTEsImlhdCI6MTc0NzA3OTg1MX0.oAS86_fHMNqzqTXcCdQqbH_c5pFtVHg9HrcJ4hy9x_0"
#}
```

### Примеры curl

#### Users API

TODO:

```shell
curl -X 'POST' 'http://localhost:8080/api/v1/register' \
  -d '{
  "login": "user",
  "password": "user"
}'
```

Ответ с кодом 200:

```json
{}
```

TODO:

```shell
curl -X 'POST' 'http://localhost:8080/api/v1/register' \
  -d '{
  "login": "user",
  "password": "user"
}'
```

Ответ с кодом 400:

```json
{
  "code": 6,
  "message": "user exists",
  "details": []
}
```

TODO:

```shell
curl -X 'POST' 'http://localhost:8080/api/v1/login' \
  -d '{
  "login": "user",
  "password": "user"
}'
```

Ответ с кодом 200:

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2luZm8iOnsiaWQiOiJkMGg1NGVsNmhqNmM3Mzk2YWt2MCIsImxvZ2luIjoidXNlciJ9LCJzdWIiOiJkMGg1NGVsNmhqNmM3Mzk2YWt2MCIsImV4cCI6MTc0NzA4MzM2NSwiaWF0IjoxNzQ3MDc5NzY1fQ.5PXntckgeOFuplRxSGBQxLAsbtkBtLQgE_qqestYVPA"
}
```

TODO:

```shell
curl -X 'POST' 'http://localhost:8080/api/v1/login' \
  -d '{
  "login": "pippo",
  "password": "pluto"
}'
```

Ответ с кодом 400:

```json
{
  "code": 9,
  "message": "bad login or password",
  "details": []
}
```

#### Expressions API

Получить Access Token:

```shell
#TODO
ACCESS_TOKEN=
```

Отправка арифметического выражения на вычисление:

```shell
curl -X 'POST' 'http://localhost:8080/api/v1/calculate' \
  -d '{
  "expression": "2 + 2 * 2"
}'
```

Ответ с кодом 201:

```json
{
  "id": "cv5t4a3j3vq37o313p5g"
}
```

Отправка некорректного выражения:

```shell
curl -X 'POST' 'http://localhost:8080/api/v1/calculate' \
  -d '{
  "expression": "1+"
}'
```

Ответ с кодом 422:

```json
{
  "code": 3,
  "message": "invalid expression",
  "details": []
}
```

Получение информации о конкретном выражении по его идентификатору:

```shell
curl 'http://localhost:8080/api/v1/expressions/cv5t97rj3vq3pl6kh1u0'
```

Ответ с кодом 200:

```json
{
  "expression": {
    "id": "cv5t97rj3vq3pl6kh1u0",
    "expression": "2 + 2 * 2",
    "status": "EXPRESSION_STATUS_PENDING",
    "result": 0
  }
}
```

Запрос несуществующего выражения:

```shell
curl 'http://localhost:8080/api/v1/expressions/notexists'
```

Ответ с кодом 404:

```json
{
  "code": 5,
  "message": "expression not found",
  "details": []
}
```

Получение списка всех отправленных выражений:

```shell
curl 'http://localhost:8080/api/v1/expressions'
```

Ответ с кодом 200:

```json
{
  "expressions": [
    {
      "id": "cv5rfcrj3vqdpq0e15b0",
      "expression": "2 + 2*2 + (9+3+1) / 4",
      "status": "EXPRESSION_STATUS_COMPLETED",
      "result": 9.25
    },
    {
      "id": "cv5rh8bj3vqe0iomlp4g",
      "expression": "((2+2) + (2+2) + (2+2) + (2+2)) / 0",
      "status": "EXPRESSION_STATUS_FAILED",
      "result": 0
    },
    {
      "id": "cv5t97rj3vq3pl6kh1u0",
      "expression": "2 + 2 * 2",
      "status": "EXPRESSION_STATUS_PENDING",
      "result": 0
    }
  ]
}
```

Получение всех задач для конкретного выражения (полезно для отладки):

```shell
curl 'http://localhost:8080/api/v1/expressions/cv5rfcrj3vqdpq0e15b0/tasks'
```

Ответ с кодом 200:

```json
{
  "tasks": [
    {
      "id": "cv5te3jj3vq46au1kjeg",
      "expressionId": "cv5te3jj3vq46au1kjfg",
      "parentTask1Id": "",
      "parentTask2Id": "",
      "arg1": 2,
      "arg2": 2,
      "operation": "TASK_OPERATION_MULTIPLICATION",
      "operationTime": "1s",
      "status": "TASK_STATUS_PENDING",
      "result": 0,
      "expireAt": "0001-01-01T00:00:00Z",
      "createdAt": "2025-03-08T05:35:10.982839Z",
      "updatedAt": "2025-03-08T05:35:10.982839Z"
    },
    {
      "id": "cv5te3jj3vq46au1kjf0",
      "expressionId": "cv5te3jj3vq46au1kjfg",
      "parentTask1Id": "",
      "parentTask2Id": "cv5te3jj3vq46au1kjeg",
      "arg1": 2,
      "arg2": 0,
      "operation": "TASK_OPERATION_ADDITION",
      "operationTime": "1s",
      "status": "TASK_STATUS_PENDING",
      "result": 0,
      "expireAt": "0001-01-01T00:00:00Z",
      "createdAt": "2025-03-08T05:35:10.982839Z",
      "updatedAt": "2025-03-08T05:35:10.982839Z"
    }
  ]
}
```

#### Agent API

Запрос вычислительной задачи от Calculator:

```shell
curl 'http://localhost:8080/internal/task'
```

Ответ с кодом 200:

```json
{
  "task": {
    "id": "cv5rjgjj3vqe6l04c50g",
    "arg1": 1,
    "arg2": 3,
    "operation": "TASK_OPERATION_ADDITION",
    "operationTime": "10s"
  }
}
```

Запрос задачи, когда доступных задач нет:

```shell
curl 'http://localhost:8080/internal/task'
```

Ответ с кодом 404:

```json
{
  "code": 5,
  "message": "no pending tasks",
  "details": []
}
```

Отправка результата задачи обратно в Calculator:

```shell
curl -X 'POST' 'http://localhost:8080/internal/task' \
  -d '{
  "id": "cv5rjgjj3vqe6l04c50g",
  "result": 4
}'
```

Ответ с кодом 200:

```json
{}
```

Отправка результата для несуществующей задачи:

```shell
curl -X 'POST' 'http://localhost:8080/internal/task' \
  -d '{
  "id": "notexists",
  "result": 4
}'
```

Ответ с кодом 404:

```json
{
  "code": 5,
  "message": "task not found",
  "details": []
}
```

---

> Крайний срок, или дедлайн (от англ. deadline — мёртвая линия) — дата выполнения задачи или работы, определённый момент
> времени, к которому должна быть достигнута цель или задача. По истечении этого времени элемент можно считать
> просроченным (например, для рабочих проектов или школьных заданий). Если рабочие задания или проекты не завершены к
> установленному сроку, это может отрицательно повлиять на рейтинг производительности сотрудника или оценку учащегося.

🙏 Просьба не смотреть ничего, кроме актуального main.
