#***** Generation
.PHONY: gen-proto
gen-proto:
	buf generate

.PHONY: gen-mocks
gen-mocks:
	go run github.com/vektra/mockery/v2@v2.53.3

.PHONY: generate
generate: gen-proto gen-mocks

#***** DB
DB_SQLITE_PATH ?= .data/db.sqlite

.PHONY: migrate
migrate:
	mkdir -p $(dir $(DB_SQLITE_PATH)) \
	&& go run -tags "sqlite3" github.com/golang-migrate/migrate/v4/cmd/migrate@latest -database sqlite3://$(DB_SQLITE_PATH) -path migrations up

#***** Build
.PHONY: build-calculator
build-calculator:
	CGO_ENABLED=1 go build -o ./bin/calculator ./cmd/calculator

.PHONY: build-agent
build-agent:
	CGO_ENABLED=0 go build -o ./bin/agent ./cmd/agent

#***** Docker
.PHONY: up
up: migrate
	docker-compose up

#***** Lint
.PHONY: lint
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.6 run ./...

#***** Tests
.PHONY: test
test:
	go test -v ./internal/...

.PHONY: test-cov
test-cov:
	mkdir -p .coverage \
	&& go test ./internal/... -coverprofile=.coverage/cover \
	&& go tool cover -html=.coverage/cover -o .coverage/cover.html
