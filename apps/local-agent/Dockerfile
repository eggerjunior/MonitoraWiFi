# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY apps/local-agent/go.mod apps/local-agent/go.sum ./
RUN go mod download
COPY apps/local-agent/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/agent ./cmd/agent
# distroless (estágio final) não tem shell nem mkdir — /data precisa existir
# e já pertencer ao usuário nonroot ANTES do COPY, senão o agente falha ao
# persistir estado/fila com "permission denied" (achado real validando esta
# imagem ponta a ponta, não hipotético).
RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/agent /app/agent
COPY --from=build --chown=nonroot:nonroot /out/data /data
ENV STATE_FILE=/data/agent.json
ENV QUEUE_FILE=/data/queue.jsonl
ENV SPEED_QUEUE_FILE=/data/speed-queue.jsonl
ENTRYPOINT ["/app/agent"]
