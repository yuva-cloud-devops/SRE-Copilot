#!/usr/bin/env bash
set -euo pipefail

echo "=========================================="
echo " Starting SRE Copilot VM Provisioning..."
echo "=========================================="

# 1. Update OS and install prerequisites (Nginx, Postgres client, logrotate)
apt-get update -y
apt-get install -y curl wget jq ca-certificates postgresql-client nginx logrotate

# 2. Create app user, runtime directory, and dedicated log directory
USER_NAME="${USER_NAME}"
USER_HOME="/home/$USER_NAME"
BUILDBIN_DIR="$USER_HOME/build-bin"
LOG_DIR="/var/log/sre-copilot"

id -u "$USER_NAME" &>/dev/null || useradd -m -s /bin/bash "$USER_NAME"
mkdir -p "$BUILDBIN_DIR" "$LOG_DIR"

# Pre-create application log files with correct ownership
touch "$LOG_DIR/sre-copilot.log"
chown -R "$USER_NAME:$USER_NAME" "$LOG_DIR"
chmod 750 "$LOG_DIR"
chmod 640 "$LOG_DIR/sre-copilot.log"

# 3. Write production config.json directly into build-bin
cat <<EOF > "$BUILDBIN_DIR/config.json"
{
  "server": {
    "port": "8080",
    "environment": "production",
    "app_name": "SRE Copilot Monolith"
  },
  "database": {
    "database_url": "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/${DB_NAME}?sslmode=disable",
    "max_open_conns": 25,
    "max_idle_conns": 5,
    "conn_max_lifetime_minutes": 30
  },
  "storage": {
    "bucket_name": "${STORAGE_BUCKET}"
  },
  "gcp": {
    "project_id": "${PROJECT_ID}",
    "default_region": "${DEFAULT_REGION}",
    "alerting_webhook_path": "/api/v1/webhooks/gcp/monitoring",
    "logging_webhook_path": "/api/v1/webhooks/gcp/logging"
  },
  "ai": {
    "provider": "gemini",
    "gemini_api_key": "",
    "model_name": "gemini-1.5-pro",
    "simulate_token_delay_ms": 15
  },
  "correlation": {
    "sliding_window_minutes": 15,
    "auto_group_by_service": true,
    "deduplicate_by_fingerprint": true
  }
}
EOF

# Fix ownership and permissions for runtime directory
chown -R "$USER_NAME:$USER_NAME" "$USER_HOME"
chmod 755 "$BUILDBIN_DIR"
chmod 600 "$BUILDBIN_DIR/config.json"

# 4. Grant sudo permissions for GitHub Actions CI/CD to restart services without password
cat <<EOF > /etc/sudoers.d/srecopilot-deploy
${USER_NAME} ALL=(ALL) NOPASSWD: /bin/systemctl restart sre-copilot, /bin/systemctl status sre-copilot, /bin/systemctl start sre-copilot, /bin/systemctl stop sre-copilot, /bin/systemctl restart nginx, /bin/systemctl reload nginx, /bin/systemctl status nginx
EOF
chmod 440 /etc/sudoers.d/srecopilot-deploy

# 5. Configure Automated Log Rotation (14-day retention with copytruncate)
cat <<EOF > /etc/logrotate.d/sre-copilot
/var/log/sre-copilot/*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 ${USER_NAME} ${USER_NAME}
    copytruncate
}
EOF

# 6. Create Systemd Service writing application logs directly to /var/log/sre-copilot/
cat <<EOF > /etc/systemd/system/sre-copilot.service
[Unit]
Description=SRE Copilot Incident Intelligence Monolith
After=network.target

[Service]
Type=simple
User=${USER_NAME}
WorkingDirectory=/home/${USER_NAME}/build-bin
ExecStart=/home/${USER_NAME}/build-bin/sre-copilot
Restart=always
RestartSec=5s
Environment=PORT=8080
Environment=GIN_MODE=release

# Application Log Persistence on Server
StandardOutput=append:/var/log/sre-copilot/sre-copilot.log
StandardError=append:/var/log/sre-copilot/sre-copilot.log
ReadWritePaths=/var/log/sre-copilot

# Security sandboxing
NoNewPrivileges=true
ProtectSystem=full

[Install]
WantedBy=multi-user.target
EOF

# 7. Configure Nginx Reverse Proxy (Port 80 -> Port 8080 with SSE streaming support & logs)
cat <<'EOF' > /etc/nginx/sites-available/sre-copilot.conf
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;

    client_max_body_size 20M;

    # Nginx access and error logs
    access_log /var/log/nginx/sre-copilot-access.log combined;
    error_log /var/log/nginx/sre-copilot-error.log warn;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;

        # Standard reverse proxy headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Critical for Server-Sent Events (SSE) live timeline & AI streaming tokens
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_cache off;
        chunked_transfer_encoding off;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }
}
EOF

# Enable Nginx site and remove default site
rm -f /etc/nginx/sites-enabled/default
ln -sf /etc/nginx/sites-available/sre-copilot.conf /etc/nginx/sites-enabled/

# Test Nginx syntax and reload
nginx -t
systemctl restart nginx
systemctl enable nginx
systemctl daemon-reload
systemctl enable sre-copilot.service

echo "Nginx reverse proxy configured: Port 80 -> http://127.0.0.1:8080"
echo "Runtime directory configured: $BUILDBIN_DIR"
echo "Application logs configured:  $LOG_DIR/sre-copilot.log"
echo "Log rotation configured:      /etc/logrotate.d/sre-copilot"
echo "Ready for GitHub Actions CI/CD artifact deployment."
echo "=========================================="
echo " VM Provisioning Complete!"
echo "=========================================="
