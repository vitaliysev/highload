FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-s -w" \
        -o /bin/worker \
        ./cmd/

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/worker /bin/worker

ENTRYPOINT ["/bin/worker"]
