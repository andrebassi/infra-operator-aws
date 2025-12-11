# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /workspace

# Copy go mod files first (better caching)
COPY go.mod go.sum ./

# Download dependencies (cached layer)
RUN go mod download

# Copy source code
COPY main.go ./
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY internal/ internal/

# Build - removed -a flag, added ldflags for smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o infra.operator main.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/infra.operator .

USER 65532:65532
ENTRYPOINT ["/infra.operator"]
