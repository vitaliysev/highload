FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-s -w" \
        -o /bin/listing-service \
        ./cmd/

FROM alpine:3.19

RUN apk add --no-cache ca-certificates wget

COPY --from=builder /bin/listing-service /bin/listing-service

EXPOSE 8080

ENTRYPOINT ["/bin/listing-service"]
