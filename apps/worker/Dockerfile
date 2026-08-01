# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY apps/worker/go.mod apps/worker/go.sum ./
RUN go mod download
COPY apps/worker/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=build /out/worker /app/worker
USER nonroot:nonroot
ENTRYPOINT ["/app/worker"]
