FROM golang:1.23-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o calculator ./cmd/calculator
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o agent ./cmd/agent

FROM gcr.io/distroless/static:nonroot AS prod

WORKDIR /

COPY --from=builder /app/calculator .
COPY --from=builder /app/agent .

USER 65532:65532
