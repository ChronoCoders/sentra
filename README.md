# Sentra v2

Sentra is a lightweight, secure WireGuard management system. Version 2 introduces a decoupled architecture with a local event bus for efficient status updates and improved scalability.

## Architecture

Sentra v2 follows a modular architecture:

-   **Control Plane**: The central management server.
-   **Agent**: A lightweight component that monitors WireGuard interface status.
-   **Event Bus**: A local in-memory message bus that decouples the Agent from the Control Plane.
-   **Status Cache**: An in-memory cache that provides instant status updates to the API without blocking on WireGuard kernel calls.

### Key Components

-   **Control Plane (`cmd/control`)**:
    -   Manages the API and database.
    -   Initializes the `EventBus` and `StatusCache`.
    -   Embeds the Agent for local monitoring.
-   **Agent (`internal/agent`)**:
    -   Periodically polls WireGuard interface status (every 10s).
    -   Publishes `StatusEvent` to the `EventBus`.
-   **API (`internal/api`)**:
    -   Serves status information from the `StatusCache`.
    -   Authentication via JWT.
-   **Storage**: SQLite (via `modernc.org/sqlite`).

## Getting Started

### Prerequisites

-   Go 1.22+
-   WireGuard tools installed (for `wg` command and kernel module).

### Installation

1.  Clone the repository:
    ```bash
    git clone https://github.com/ChronoCoders/sentra.git
    cd sentra
    ```

2.  Build the binaries:
    ```bash
    go build -o sentra-control ./cmd/control
    go build -o sentra-agent ./cmd/agent
    ```

### Configuration

Sentra requires environment variables for configuration. See `.env.example` (if available) or check `internal/config/config.go`.

Key variables:
-   `WG_INTERFACE`: WireGuard interface name (default: `wg0`).
-   `PORT`: API server port (default: `8080`).

### Running

Run the control plane (which includes the local agent):

```bash
./sentra-control
```

## Development Status

-   [x] **Core Architecture**: Control/Agent split, EventBus, StatusCache.
-   [x] **WireGuard Integration**: `wgctrl-go` for interface management.
-   [x] **API**: Basic status endpoint reading from cache.
-   [ ] **Authentication**: JWT middleware integration (in progress).
-   [ ] **Remote Agents**: Support for remote agents via gRPC/HTTP (future).

## License

MIT
