# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.mod
COPY go.sum go.sum*

# Copy the source code
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY internal/ internal/

# Download dependencies and tidy
ENV GOTOOLCHAIN=auto
RUN go mod download && go mod tidy

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -o infra.operator main.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /

# Copy the binary from builder
COPY --from=builder /workspace/infra.operator .

USER 65532:65532

ENTRYPOINT ["/infra.operator"]
