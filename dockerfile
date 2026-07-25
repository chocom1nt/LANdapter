FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bin/master cmd/master/main.go
RUN go build -o bin/agent cmd/agent/main.go

FROM debian:bullseye-slim
WORKDIR /app
COPY --from=builder /app/bin ./bin
CMD ["./bin/master"]