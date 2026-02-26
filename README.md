# Sentra v1.0

Sentra v1.0 is a simple, lightweight web dashboard for monitoring WireGuard interfaces managed by [wg-easy](https://github.com/wg-easy/wg-easy).

> **Note**: This is the legacy v1.0 branch. For the latest features and decoupled architecture, please switch to the `main` branch (Sentra v2).

## Features

-   **Real-time Dashboard**: Visualizes WireGuard status, peer handshakes, and transfer rates.
-   **Docker Integration**: Directly queries the `wg-easy` container for interface statistics.
-   **Metrics**: Exposes Prometheus metrics at `/metrics`.
-   **REST API**: Simple JSON API for status and health checks.

## Architecture

Sentra v1.0 is a monolithic Go application that:
1.  Serves a static frontend (`frontend/`).
2.  Exposes an API (`backend/`).
3.  Executes `docker exec` commands to retrieve WireGuard status from a running `wg-easy` container.

## Prerequisites

-   Go 1.23+
-   Docker
-   A running `wg-easy` container named `wg-easy`.

## Getting Started

### Running Locally

1.  Ensure your `wg-easy` container is running:
    ```bash
    docker ps | grep wg-easy
    ```

2.  Navigate to the backend directory:
    ```bash
    cd backend
    ```

3.  Run the application:
    ```bash
    go run .
    ```
    The server will start on `127.0.0.1:8080`.

### Building

```bash
cd backend
go build -o sentra-v1
./sentra-v1
```

## API Endpoints

-   `GET /api/status`: Returns current WireGuard interface and peer status.
-   `GET /api/health`: Health check endpoint.
-   `GET /api/events`: (Experimental) Event stream.
-   `GET /api/logs`: Retrieve application logs.
-   `GET /metrics`: Prometheus metrics.

## License

MIT
