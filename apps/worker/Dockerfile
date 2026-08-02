# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY apps/worker/go.mod apps/worker/go.sum ./
RUN go mod download
COPY apps/worker/ ./

# Versionamento (skill ildemar_app-versioning): VERSION é a fonte única,
# GIT_COMMIT é injetado pelo processo de release — nunca hardcoded.
ARG GIT_COMMIT=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X egger/worker/internal/buildinfo.Version=$(cat VERSION) -X egger/worker/internal/buildinfo.GitCommit=${GIT_COMMIT}" -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=build /out/worker /app/worker
USER nonroot:nonroot
ENTRYPOINT ["/app/worker"]
