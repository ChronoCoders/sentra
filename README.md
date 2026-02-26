# Sentra v2

Sentra is a lightweight, secure WireGuard management system. Version 2 introduces a decoupled architecture with a local event bus for efficient status updates and improved scalability.

## Architecture

Sentra v2 follows a modular architecture:

-   **Control Plane**: The central management server.
-   **Agent**: A lightweight component that monitors WireGuard interface status and system metrics.
-   **Event Bus**: A local in-memory message bus that decouples the Agent from the Control Plane.
-   **Status Cache**: An in-memory cache that provides instant status updates to the API without blocking on WireGuard kernel calls.

### Key Components

-   **Control Plane (`cmd/control`)**:
    -   Manages the API, database, and WebSocket connections.
    -   Initializes the `EventBus` and `StatusCache`.
    -   Embeds the Agent for local monitoring.
-   **Agent (`internal/agent`)**:
    -   Periodically polls WireGuard interface status and system metrics (every 10s).
    -   Publishes `StatusEvent` to the `EventBus` or via HTTP to a remote Control Plane.
-   **API (`internal/api`)**:
    -   Serves status information from the `StatusCache`.
    -   Authentication via JWT.
    -   Real-time updates via WebSocket.
-   **Storage**: SQLite (via `modernc.org/sqlite`).

## Getting Started

### Prerequisites

-   Go 1.22+ (for building from source)
-   **WireGuard Kernel Module & Tools**: Sentra strictly requires a real WireGuard interface (e.g., `wg0`). Mock mode has been removed.
-   Docker & Docker Compose (recommended for deployment)

### Installation (Docker)

1.  Clone the repository:
    ```bash
    git clone https://github.com/ChronoCoders/sentra.git
    cd sentra
    ```

2.  Run with Docker Compose:
    ```bash
    docker-compose up -d --build
    ```

    **Note**: The Agent requires host network mode to access the WireGuard interface. Ensure your `docker-compose.yml` is configured correctly.

### Installation (Manual)

1.  Build the binaries:
    ```bash
    go build -o sentra-control ./cmd/control
    go build -o sentra-agent ./cmd/agent
    ```

2.  Run the control plane (which includes the local agent):
    ```bash
    sudo ./sentra-control
    ```

### Configuration

Sentra requires environment variables for configuration. See `.env.example` (if available) or check `internal/config/config.go`.

Key variables:
-   `WG_INTERFACE`: WireGuard interface name (default: `wg0`).
-   `PORT`: API server port (default: `8080`).
-   `JWT_SECRET`: Secret key for JWT authentication.

### SSL Configuration

Sentra supports HTTPS out of the box. You can either provide your own certificates or let Sentra generate self-signed certificates for local development.

Key variables:
-   `SENTRA_TLS_CERT`: Path to the TLS certificate file.
-   `SENTRA_TLS_KEY`: Path to the TLS private key file.
-   `SENTRA_TLS_AUTO`: Set to `true` to automatically generate self-signed certificates if `SENTRA_TLS_CERT` and `SENTRA_TLS_KEY` are not provided or do not exist. (Default: `false`)

Example (Docker Compose):
```yaml
environment:
  - SENTRA_TLS_AUTO=true
```

### Agent Configuration (SSL)

If you are using self-signed certificates on the Control Plane, you must configure the Agent to skip verification:

```yaml
environment:
  - SENTRA_INSECURE_SKIP_VERIFY=true
```

## Features & Status

-   [x] **Core Architecture**: Control/Agent split, EventBus, StatusCache.
-   [x] **WireGuard Integration**: `wgctrl-go` for interface management (Real interface required).
-   [x] **System Metrics**: CPU, Memory, Disk, Load usage monitoring.
-   [x] **API**: REST API for status and management.
-   [x] **Authentication**: JWT middleware and Role-Based Access Control (RBAC).
-   [x] **Real-time Dashboard**: WebSocket-based live updates for traffic and peer status.
-   [x] **Remote Agents**: Support for remote agents via HTTP reporting.

## License

MIT
