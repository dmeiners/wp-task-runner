FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go modules first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build fully static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -ldflags="-s -w -extldflags '-static'" -trimpath \
    -o /wp-task-runner ./cmd/daemon

# Final stage - truly minimal
FROM scratch

COPY --from=builder /wp-task-runner /wp-task-runner

# You can copy default config if desired, but better to mount it
# COPY config.yaml.example /etc/wp-task-runner/config.yaml

ENTRYPOINT ["/wp-task-runner"]
