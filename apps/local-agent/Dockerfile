# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY apps/local-agent/go.mod apps/local-agent/go.sum ./
RUN go mod download
COPY apps/local-agent/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/agent ./cmd/agent

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/agent /app/agent
ENV STATE_FILE=/data/agent.json
ENV QUEUE_FILE=/data/queue.jsonl
ENTRYPOINT ["/app/agent"]
