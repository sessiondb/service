FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# BUILD_TAGS: set to "pro" for premium image; leave empty or omit for community.
ARG BUILD_TAGS
RUN if [ -n "$BUILD_TAGS" ]; then \
      CGO_ENABLED=0 GOOS=linux go build -tags "$BUILD_TAGS" -o sessiondb ./cmd/server; \
    else \
      CGO_ENABLED=0 GOOS=linux go build -o sessiondb ./cmd/server; \
    fi

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/sessiondb .

EXPOSE 8080

CMD ["./sessiondb"]
