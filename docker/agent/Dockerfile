FROM golang:1.25-alpine AS builder

WORKDIR /agent
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o agent cmd/agent/main.go

FROM alpine:latest

WORKDIR /root
COPY --from=builder /agent .
CMD ["./agent"]
