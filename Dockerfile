FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for go mod download if needed
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build Control Plane
RUN CGO_ENABLED=0 go build -o control ./cmd/control

# Build Agent
RUN CGO_ENABLED=0 go build -o agent ./cmd/agent

# Final Stage
FROM alpine:latest

WORKDIR /app

# Install WireGuard tools (wg-quick, wg) for Agent if needed (though we use userspace/kernel via netlink, tools are helpful for debug)
RUN apk add --no-cache wireguard-tools iproute2 openssl

COPY --from=builder /app/control .
COPY --from=builder /app/agent .
COPY --from=builder /app/web ./web

# Expose ports
EXPOSE 8080
EXPOSE 51820/udp

CMD ["./control"]
