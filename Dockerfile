FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_Enabled=0 GOOS=linux go build -o sessiondb ./cmd/server

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/sessiondb .

EXPOSE 8080

CMD ["./sessiondb"]
