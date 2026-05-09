FROM golang:1.25-alpine AS builder

WORKDIR /scrapper
COPY go.mod go.sum ./
RUN go mod tidy
RUN go mod download
COPY . .
RUN go build -o scrapper cmd/scrapper/main.go

FROM alpine:latest

WORKDIR /root
COPY --from=builder /scrapper .
COPY config.yaml config.yaml
CMD ["./scrapper"]
