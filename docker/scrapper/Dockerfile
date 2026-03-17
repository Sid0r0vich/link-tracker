FROM golang:1.25-alpine AS builder

WORKDIR /scrapper
COPY . .
RUN go mod tidy
RUN go mod download
RUN go build -o scrapper cmd/scrapper/main.go

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /scrapper .
CMD ["./scrapper"]
