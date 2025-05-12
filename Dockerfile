FROM golang:1.24 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o calculator ./cmd/calculator
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o agent ./cmd/agent

RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -tags "sqlite3" -o migrate github.com/golang-migrate/migrate/v4/cmd/migrate

FROM gcr.io/distroless/base:nonroot AS prod

WORKDIR /

COPY --from=builder /app/calculator .
COPY --from=builder /app/agent .

COPY --from=builder /app/migrate .

USER 65532:65532
