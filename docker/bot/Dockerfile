FROM golang:1.25-alpine AS builder

WORKDIR /bot
COPY . .
RUN go mod tidy
RUN go mod download
RUN go build -o bot cmd/bot/main.go

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /bot .
CMD ["./bot"]
