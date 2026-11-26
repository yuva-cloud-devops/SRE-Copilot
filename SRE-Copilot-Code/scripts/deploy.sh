#!/usr/bin/env bash
set -euo pipefail

echo "==> Staging files to /home/srecopilot/build-bin/ ..."
mkdir -p /home/srecopilot/build-bin/templates /home/srecopilot/build-bin/static

# 1. Update binary
if [ -f /tmp/deploy-bundle/sre-copilot ]; then
  cp -f /tmp/deploy-bundle/sre-copilot /home/srecopilot/build-bin/sre-copilot
  chmod +x /home/srecopilot/build-bin/sre-copilot
fi

# 2. Sync templates and static assets
if [ -d /tmp/deploy-bundle/templates ]; then
  cp -rf /tmp/deploy-bundle/templates/* /home/srecopilot/build-bin/templates/
fi
if [ -d /tmp/deploy-bundle/static ]; then
  cp -rf /tmp/deploy-bundle/static/* /home/srecopilot/build-bin/static/
fi

# 3. Synchronize config.json
if [ -f /tmp/deploy-bundle/config.json ]; then
  # If server already has a gemini_api_key set and the incoming bundle key is empty, preserve existing key
  if [ -f /home/srecopilot/build-bin/config.json ]; then
    EXISTING_KEY=$(jq -r '.ai.gemini_api_key // empty' /home/srecopilot/build-bin/config.json 2>/dev/null || true)
    INCOMING_KEY=$(jq -r '.ai.gemini_api_key // empty' /tmp/deploy-bundle/config.json 2>/dev/null || true)
    if [ -n "$EXISTING_KEY" ] && [ -z "$INCOMING_KEY" ]; then
      jq --arg key "$EXISTING_KEY" '.ai.gemini_api_key = $key' /tmp/deploy-bundle/config.json > /tmp/deploy-bundle/config.tmp && mv /tmp/deploy-bundle/config.tmp /tmp/deploy-bundle/config.json
    fi
  fi
  cp -f /tmp/deploy-bundle/config.json /home/srecopilot/build-bin/config.json
  chmod 600 /home/srecopilot/build-bin/config.json
fi

# 4. Enforce proper ownership
chown -R srecopilot:srecopilot /home/srecopilot/build-bin

# 5. Direct systemd logs to journald so 'sudo journalctl -u sre-copilot -f' works in real time
if [ -f /etc/systemd/system/sre-copilot.service ]; then
  if grep -q "StandardOutput=append:" /etc/systemd/system/sre-copilot.service; then
    sed -i 's|StandardOutput=append:.*|StandardOutput=journal|' /etc/systemd/system/sre-copilot.service
    sed -i 's|StandardError=append:.*|StandardError=journal|' /etc/systemd/system/sre-copilot.service
    systemctl daemon-reload
    echo "✓ Reconfigured sre-copilot.service logging to journald"
  fi
fi

# 6. Clean up temporary staging directory
rm -rf /tmp/deploy-bundle

# 7. Restart all services
echo "==> Restarting SRE Copilot and Nginx services..."
systemctl restart sre-copilot
systemctl restart nginx

# 8. Check service health
echo "==> Checking service health..."
systemctl is-active --quiet sre-copilot && echo "✓ sre-copilot is ACTIVE"
systemctl is-active --quiet nginx && echo "✓ nginx is ACTIVE"
