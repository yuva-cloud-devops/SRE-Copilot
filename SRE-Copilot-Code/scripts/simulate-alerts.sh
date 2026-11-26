#!/usr/bin/env bash
# ==============================================================================
# SRE Copilot - End-to-End Alert Simulation Script
# Sends realistic Google Cloud Monitoring and Cloud Logging payloads to the SRE Copilot webhook
# ==============================================================================

set -euo pipefail

TARGET_HOST="${1:-http://34.47.244.171}"
echo "Targeting SRE Copilot at: ${TARGET_HOST}"

MONITORING_URL="${TARGET_HOST}/api/v1/webhooks/gcp/monitoring"
LOGGING_URL="${TARGET_HOST}/api/v1/webhooks/gcp/logging"

echo ""
echo "=========================================================="
echo "1. Simulating GCP Cloud Monitoring Alert: Cloud Run 5xx Spike"
echo "=========================================================="
curl -s -X POST "${MONITORING_URL}" \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1.2",
    "incident": {
      "incident_id": "0.m89abc123def456-5xx",
      "resource_id": "sample-nginx-service-dfe66177",
      "resource_name": "sample-nginx-service-dfe66177",
      "resource": {
        "type": "cloud_run_revision",
        "labels": {
          "service_name": "sample-nginx-service",
          "project_id": "yuvaraj-internal-project",
          "location": "asia-south1"
        }
      },
      "policy_name": "Cloud Run 5xx Server Errors Detected",
      "condition_name": "Cloud Run request count with response_code_class = 5xx > 5 req/min",
      "url": "https://console.cloud.google.com/monitoring/alerting/incidents/0.m89abc123def456",
      "state": "open",
      "started_at": '$(date +%s)',
      "summary": "Cloud Run service sample-nginx-service is experiencing sustained HTTP 500 internal server errors."
    }
  }' | jq . || true

echo ""
sleep 2

echo "=========================================================="
echo "2. Simulating GCP Cloud Monitoring Alert: Cloud Run Latency Saturation"
echo "=========================================================="
curl -s -X POST "${MONITORING_URL}" \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1.2",
    "incident": {
      "incident_id": "0.m89abc123def456-latency",
      "resource_id": "sample-nginx-service-dfe66177",
      "resource_name": "sample-nginx-service-dfe66177",
      "resource": {
        "type": "cloud_run_revision",
        "labels": {
          "service_name": "sample-nginx-service",
          "project_id": "yuvaraj-internal-project",
          "location": "asia-south1"
        }
      },
      "policy_name": "Cloud Run p95 Latency Degradation (>2000ms)",
      "condition_name": "95th percentile latency > 2s over 2 minutes",
      "url": "https://console.cloud.google.com/monitoring/alerting/incidents/0.m89abc123def456-latency",
      "state": "open",
      "started_at": '$(date +%s)',
      "summary": "Cloud Run service sample-nginx-service p95 request latency degraded to 3420ms."
    }
  }' | jq . || true

echo ""
sleep 2

echo "=========================================================="
echo "3. Simulating Cloud Logging Log Sink: Fatal DB Error"
echo "=========================================================="
curl -s -X POST "${LOGGING_URL}" \
  -H "Content-Type: application/json" \
  -d '{
    "insertId": "log-entry-uuid-998877",
    "severity": "CRITICAL",
    "textPayload": "FATAL: remaining connection slots are reserved for non-replication superuser connections. Total active pool connections: 100/100.",
    "resource": {
      "type": "cloudsql_database",
      "labels": {
        "database_id": "yuvaraj-internal-project:sre-copilot-db-dfe66177",
        "project_id": "yuvaraj-internal-project",
        "region": "asia-south1"
      }
    },
    "timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'",
    "labels": {
      "service": "sample-nginx-service",
      "environment": "production"
    }
  }' | jq . || true

echo ""
echo "=========================================================="
echo "✓ Alert Simulation Complete!"
echo "Check your SRE Copilot Dashboard at: ${TARGET_HOST}"
echo "=========================================================="
