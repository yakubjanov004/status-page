# Self-Hosted Status Page

A lightweight, self-hosted uptime monitoring and public status page system. Built with Go (Golang), SQLite, Vanilla JS, and Vanilla CSS.

## Features
- **Auto-Discovery**: Scans Nginx and Systemd files to automatically find and monitor local projects!
- **Project Grouping**: Groups Frontend and Backend components logically.
- **HTTP/HTTPS & TCP Monitoring**: Monitor your API, website, or database.
- **Public Status Page**: A beautiful, modern status page showing 7-day uptime history per component, computed client-side via the incidents API using the /api/v1/ nginx proxy topology.
- **Real-time Updates**: WebSocket integration pushes state changes to the UI without refreshing.
- **Zero Dependencies**: Uses a pure Go SQLite driver (`modernc.org/sqlite`), no CGO required.
- **Webhook Receiver**: Accept service up/down events via HTTP webhooks for incident tracking.
- **Docker Watcher**: Monitor Docker container lifecycle events (Tokpoint).
- **systemd Integration**: Automatic notifications via drop-in configuration files.

---

## Installation (Status Page — Systemd + Nginx)

This project is designed to run directly on your server via `systemd`.

1. **Build the binary:**
   ```bash
   go mod tidy
   go build -o status-page ./cmd/server
   ```
2. **Move to /opt/status-page:**
   ```bash
   sudo mkdir -p /opt/status-page
   sudo cp status-page /opt/status-page/
   sudo cp -r web /opt/status-page/web
   sudo cp .env.example /opt/status-page/.env
   # Edit /opt/status-page/.env to set your passwords and tokens
   ```
3. **Setup Systemd Service:**
   ```bash
   sudo cp deploy/status-page.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable --now status-page
   ```
4. **Nginx Reverse Proxy:**
   Check the `deploy/nginx-status-page.conf` file as a starting point.
   ```bash
   sudo cp deploy/nginx-status-page.conf /etc/nginx/sites-available/status-page
   sudo ln -s /etc/nginx/sites-available/status-page /etc/nginx/sites-enabled/
   sudo systemctl reload nginx
   ```

---

## Webhook Receiver

The webhook receiver is a separate lightweight HTTP service that records service up/down events, manages incidents, computes uptime, and provides a JSON API.

### Building

```bash
# Build the webhook receiver binary
go build -o status-webhook ./cmd/webhook

# Build the Docker watcher binary
go build -o docker-watcher ./cmd/dockerwatch
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `HOOK_TOKEN` | *(required)* | Secret token for `X-Hook-Token` header authentication |
| `DATABASE_PATH` | `./data/status.db` | Path to the SQLite database file |
| `LISTEN_ADDR` | `:8080` | Address and port to listen on |
| `TOKPOINT_CONTAINER_NAME` | `tokpoint` | Docker container name to watch |
| `TOKPOINT_SERVICE_NAME` | `Tokpoint` | Service name to use in webhooks |
| `WEBHOOK_URL` | `http://127.0.0.1:8080/api/v1/webhook` | Webhook endpoint URL (used by docker-watcher and service-notify.sh) |

### Deploying the Webhook Receiver

```bash
# 1. Install binary
sudo cp status-webhook /usr/local/bin/status-webhook

# 2. Create working directory
sudo mkdir -p /opt/status-webhook
sudo chown statuspage:statuspage /opt/status-webhook

# 3. Create environment file
sudo tee /opt/status-webhook/.env > /dev/null <<'EOF'
HOOK_TOKEN=your-secret-token-here
DATABASE_PATH=/opt/status-webhook/data/status.db
LISTEN_ADDR=:8080
WEBHOOK_URL=http://127.0.0.1:8080/api/v1/webhook
TOKPOINT_CONTAINER_NAME=tokpoint
EOF

# 4. Install and start systemd service
sudo cp deploy/status-webhook.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now status-webhook
```

### API Endpoints

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| `POST` | `/api/v1/webhook` | `X-Hook-Token` | Record a service event |
| `GET` | `/api/v1/services` | — | List all services with status |
| `GET` | `/api/v1/services/{name}/incidents` | — | List incidents (paginated) |
| `GET` | `/api/v1/services/{name}/uptime?window=24h\|7d\|30d` | — | Uptime percentage |
| `GET` | `/healthz` | — | Health check |
| `GET` | `/status` | — | Simple HTML status page |

### Example Curl Commands

```bash
# Send a "down" event
curl -X POST http://localhost:8080/api/v1/webhook \
  -H "Content-Type: application/json" \
  -H "X-Hook-Token: your-secret-token-here" \
  -d '{
    "service": "AlfaConnect",
    "action": "down",
    "time": "2025-06-15T10:00:00Z",
    "meta": {"reason": "process crashed", "unit": "alfaconnect-bot.service"}
  }'

# Send an "up" event
curl -X POST http://localhost:8080/api/v1/webhook \
  -H "Content-Type: application/json" \
  -H "X-Hook-Token: your-secret-token-here" \
  -d '{
    "service": "AlfaConnect",
    "action": "up",
    "time": "2025-06-15T10:05:00Z",
    "meta": {"unit": "alfaconnect-bot.service"}
  }'

# List all services
curl http://localhost:8080/api/v1/services | jq .

# List incidents for a service
curl "http://localhost:8080/api/v1/services/AlfaConnect/incidents?limit=10&offset=0" | jq .

# Get 7-day uptime
curl "http://localhost:8080/api/v1/services/AlfaConnect/uptime?window=7d" | jq .

# Health check
curl http://localhost:8080/healthz
```

### Monitored Services

The following services are pre-seeded in the database on startup:

| Service Name | Integration Method |
|---|---|
| **AlfaConnect** | systemd drop-in (alfaconnect-bot.service, alfaconnect-webapp.service) |
| **Mehmonxona** | systemd drop-in (mehmonxona.service) |
| **Odimrepo** | systemd drop-in (odimrepo-backend.service, odimrepo-frontend.service) |
| **Tokpoint** | Docker watcher + systemd drop-in for non-Docker units |
| **Datan** | systemd drop-in (datan.service) |

---

## Docker Watcher (Tokpoint)

The `docker-watcher` binary monitors Docker container lifecycle events for the Tokpoint service and translates them into webhook POST requests.

### How It Works

1. Connects to the Docker daemon via `/var/run/docker.sock`
2. Filters events for the container named `tokpoint` (configurable)
3. Maps container events:
   - `start`, `unpause`, `health_status: healthy` → sends `action: "up"`
   - `die`, `stop`, `kill`, `oom`, `pause`, `health_status: unhealthy` → sends `action: "down"`
4. POSTs to the webhook endpoint with `X-Hook-Token` authentication

### Deploying

```bash
# Install binary
sudo cp docker-watcher /usr/local/bin/docker-watcher

# Install systemd service
sudo cp deploy/dockerwatch.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now dockerwatch
```

### Requirements

- **Docker socket access**: The watcher needs read access to `/var/run/docker.sock`. The systemd unit runs as `root` by default. Alternatively, add the service user to the `docker` group:
  ```bash
  sudo usermod -aG docker statuspage
  ```
- **Container naming**: Ensure the Tokpoint container is named exactly `tokpoint`:
  ```bash
  docker run --name tokpoint ...
  # or in docker-compose.yml:
  # container_name: tokpoint
  ```
- You can also use container labels or a different name by setting `TOKPOINT_CONTAINER_NAME`.

---

## systemd Integration

### How It Works

1. `service-notify.sh` is called by systemd via `ExecStartPost` and `ExecStopPost` hooks
2. It maps the unit name to a canonical service name (e.g., `alfaconnect-bot.service` → `AlfaConnect`)
3. It sends a webhook POST to record the event

### Setup

```bash
# 1. Install the notification script
sudo cp deploy/service-notify.sh /usr/local/bin/service-notify.sh
sudo chmod +x /usr/local/bin/service-notify.sh

# 2. Install the failure notification template
sudo cp deploy/notify@.service /etc/systemd/system/notify@.service

# 3. Create environment file for the notify script
sudo tee /etc/default/service-notify > /dev/null <<'EOF'
WEBHOOK_URL=http://127.0.0.1:8080/api/v1/webhook
HOOK_TOKEN=your-secret-token-here
EOF

# 4. Generate drop-in files (dry-run first)
bash deploy/generate-dropins.sh

# 5. Generate drop-in files (apply)
bash deploy/generate-dropins.sh --apply

# 6. Install generated drop-ins (follow printed commands)
# Example:
sudo mkdir -p /etc/systemd/system/alfaconnect-bot.service.d
sudo cp generated-dropins/alfaconnect-bot.service.d/notify.conf /etc/systemd/system/alfaconnect-bot.service.d/

# 7. Reload systemd
sudo systemctl daemon-reload
```

### Manual Drop-in Example

To manually add webhook notifications to any systemd unit, create a drop-in file:

```bash
sudo mkdir -p /etc/systemd/system/myservice.service.d
sudo tee /etc/systemd/system/myservice.service.d/notify.conf > /dev/null <<'EOF'
[Service]
ExecStartPost=/usr/local/bin/service-notify.sh up %n
ExecStopPost=/usr/local/bin/service-notify.sh down %n
OnFailure=notify@%n.service
EOF
sudo systemctl daemon-reload
```

---

## Security Recommendations

- **TLS**: Always run the webhook receiver behind a TLS-terminating reverse proxy (Nginx, Caddy) in production.
- **Token strength**: Use a long, random `HOOK_TOKEN` (e.g., `openssl rand -hex 32`).
- **Body size limit**: The webhook endpoint enforces a 64KB body size limit.
- **Network isolation**: Bind `LISTEN_ADDR` to `127.0.0.1:8080` if only local services send webhooks.

## Database Backup

The SQLite database is a single file. To back it up:

```bash
# Hot backup (safe while the service is running with WAL mode)
sqlite3 /opt/status-webhook/data/status.db ".backup /tmp/status-backup.db"

# Or simply copy the file (stop the service first for consistency)
sudo systemctl stop status-webhook
cp /opt/status-webhook/data/status.db /opt/status-webhook/data/status.db.bak
sudo systemctl start status-webhook
```

## Server Security Requirements (Auto-Discovery)

To allow the Auto-Discovery feature to scan Nginx and Systemd configuration files **securely without running as root**:

1. **Create a dedicated user:**
   ```bash
   sudo useradd -r -s /bin/false statuspage
   ```

2. **Add user to groups:**
   Add the user to the `www-data` (or `nginx`) group so it can read nginx configurations, and to `systemd-journal` if needed.
   ```bash
   sudo usermod -aG www-data statuspage
   ```

3. **Set ACL permissions for Systemd files (if needed):**
   ```bash
   sudo setfacl -m u:statuspage:rx /etc/systemd/system/
   ```

4. **Update Systemd Service File:**
   Make sure your `status-page.service` specifies the user:
   ```ini
   [Service]
   User=statuspage
   Group=statuspage
   # ...
   ```

**Important**: Never run the web-facing dashboard service as `root`.

---

## Running Tests

```bash
# Run all tests
go test ./... -v

# Run only webhook handler tests
go test ./internal/handler/ -v
```

## Project Structure

```
status-page/
├── cmd/
│   ├── server/          # Main status page server
│   ├── webhook/         # Webhook receiver binary
│   └── dockerwatch/     # Docker container watcher binary
├── internal/
│   ├── api/             # Status page API handlers
│   ├── config/          # Configuration loading
│   ├── db/              # Status page database layer
│   ├── handler/         # Webhook API handlers + tests
│   ├── migrate/         # Webhook migration SQL
│   ├── model/           # Webhook data models
│   ├── webhookdb/       # Webhook database layer
│   └── ...
├── deploy/
│   ├── service-notify.sh       # systemd notification script
│   ├── generate-dropins.sh     # Drop-in file generator
│   ├── notify@.service         # Failure notification template
│   ├── status-webhook.service  # Webhook receiver unit
│   ├── dockerwatch.service     # Docker watcher unit
│   ├── status-page.service     # Main status page unit
│   └── nginx-status-page.conf  # Nginx config
└── web/                 # Frontend assets
```