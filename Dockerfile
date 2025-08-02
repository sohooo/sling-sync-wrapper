# Build stage
# Build stage uses Go 1.24 to match go.mod
FROM golang:1.24 as builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/sling-sync-wrapper ./cmd/wrapper

# Runtime stage
FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/sling-sync-wrapper /sling-sync-wrapper
ENTRYPOINT ["/sling-sync-wrapper"]

