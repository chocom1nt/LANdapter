FROM golang:1.26.10 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/master ./cmd/master
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/agent ./cmd/agent
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/lndp ./cmd/lndp

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /out/master ./bin/master
COPY --from=builder /out/agent ./bin/agent
COPY --from=builder /out/lndp ./bin/lndp
COPY configs/master.yaml.example ./configs/master.yaml
COPY configs/agent.yaml.example ./configs/agent.yaml
RUN mkdir -p /app/uploads && chmod 700 /app/uploads
EXPOSE 8080 8081
CMD ["./bin/master"]
