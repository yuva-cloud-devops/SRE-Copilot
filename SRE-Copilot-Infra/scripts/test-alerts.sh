#!/usr/bin/env bash
# ==============================================================================
# SRE Copilot Alert Testing Suite
# Easily test real GCP alerts and simulated webhooks against SRE Copilot
# ==============================================================================

set -euo pipefail

TARGET_HOST="${1:-http://127.0.0.1}"
TARGET_HOST="${TARGET_HOST%/}"

SAMPLE_SERVICE_URL="${2:-}"
SAMPLE_SERVICE_URL="${SAMPLE_SERVICE_URL%/}"

echo "=========================================================="
echo " SRE Copilot Alert Testing Suite"
echo " Target Endpoint:      ${TARGET_HOST}"
if [ -n "${SAMPLE_SERVICE_URL}" ]; then
  echo " Cloud Run Sample URL: ${SAMPLE_SERVICE_URL}"
fi
echo "=========================================================="

show_menu() {
    echo ""
    echo "Select a test to trigger:"
    echo "  --- SIMULATED WEBHOOKS (0s wait, instant UI verification) ---"
    echo "  1) [INSTANT] Send simulated GCP Cloud Monitoring Alert (5xx spike)"
    echo "  2) [INSTANT] Send simulated GCP Cloud Logging Error Log (DB Connection Refused)"
    echo "  3) [INSTANT] Send an Alert Storm (5 correlated alerts in 3 seconds)"
    echo "  --- REAL GCP METRIC & LOG ALERTS (Evaluated by Google Cloud) ---"
    echo "  4) [REAL GCP] Trigger real GCP Log Alert via 'gcloud logging write'"
    echo "  5) [REAL GCP] Trigger real GCP VM CPU Metric Alert (burns VM CPU for 130s)"
    echo "  6) [REAL GCP] Trigger real Cloud Run 4xx Metric Alert (sends 15x HTTP 404 to Nginx)"
    echo "  7) [REAL GCP] Trigger real Cloud Run 5xx Metric Alert (sends 10x HTTP 500)"
    echo "  q) Quit"
    echo ""
    read -p "Enter choice [1-7 or q]: " choice
}

test_monitoring_webhook() {
    echo ""
    echo "==> Sending simulated GCP Cloud Monitoring metric alert..."
    curl -s -X POST "${TARGET_HOST}/api/v1/webhooks/gcp/monitoring" \
      -H "Content-Type: application/json" \
      -d '{
        "version": "1.2",
        "incident": {
          "incident_id": "test-gcp-alert-'$(date +%s)'",
          "resource_id": "projects/test-proj/services/sample-nginx-service",
          "resource_name": "sample-nginx-service",
          "resource": {
            "type": "cloud_run_revision",
            "labels": {
              "service_name": "sample-nginx-service",
              "environment": "production",
              "configuration_name": "sample-nginx-service-prod",
              "location": "us-central1"
            }
          },
          "metric": {
            "type": "run.googleapis.com/request_count",
            "displayName": "Request Count"
          },
          "started_at": '$(date +%s)',
          "state": "open",
          "summary": "Cloud Run 4xx client error rate spiked to 24.5% on sample-nginx-service",
          "policy_name": "Warning: Cloud Run 4xx Client Error Rate Spike",
          "condition_name": "Cloud Run 4xx Responses Count > 10",
          "url": "https://console.cloud.google.com/monitoring"
        }
      }' | jq . || true
    echo "✓ Check your SRE Copilot Dashboard at: ${TARGET_HOST}/incidents"
}

test_logging_webhook() {
    echo ""
    echo "==> Sending simulated GCP Cloud Logging error log..."
    curl -s -X POST "${TARGET_HOST}/api/v1/webhooks/gcp/logging" \
      -H "Content-Type: application/json" \
      -d '{
        "logName": "projects/test-proj/logs/sample-nginx-app",
        "severity": "ERROR",
        "timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'",
        "resource": {
          "type": "cloud_run_revision",
          "labels": {
            "service_name": "sample-nginx-service",
            "environment": "production"
          }
        },
        "textPayload": "FATAL: dial tcp 10.10.0.15:5432: connect: connection refused - PostgreSQL max_connections reached"
      }' | jq . || true
    echo "✓ Correlated log attached to active incident timeline!"
}

test_alert_storm() {
    echo ""
    echo "==> Simulating an Alert Storm (5 alerts from multiple services in rapid succession)..."
    SERVICES=("sample-nginx-service" "checkout-service" "sample-nginx-service" "auth-service" "sample-nginx-service")
    ALERTS=(
        "Warning: Cloud Run 4xx Client Error Rate Spike"
        "Critical: Cloud Run 5xx Server Error Rate Spike"
        "Warning: Cloud Run Request Latency Degradation"
        "Critical: Infrastructure & Application Error Logs"
        "Critical: Compute Engine High CPU Saturation"
    )

    for i in "${!SERVICES[@]}"; do
        SVC="${SERVICES[$i]}"
        ALT="${ALERTS[$i]}"
        echo "  -> Dispatching Alert $((i+1))/5: [${SVC}] ${ALT}..."
        curl -s -X POST "${TARGET_HOST}/api/v1/webhooks/gcp/monitoring" \
          -H "Content-Type: application/json" \
          -d '{
            "version": "1.2",
            "incident": {
              "incident_id": "storm-alert-'$i'-'$(date +%s)'",
              "resource_name": "'$SVC'",
              "resource": {
                "type": "cloud_run_revision",
                "labels": { "service_name": "'$SVC'", "environment": "production" }
              },
              "metric": { "type": "run.googleapis.com/request_count" },
              "started_at": '$(date +%s)',
              "state": "open",
              "summary": "Alert Storm Test: '$ALT' on '$SVC'",
              "policy_name": "'$ALT'",
              "condition_name": "Threshold Breached",
              "url": "https://console.cloud.google.com/monitoring"
            }
          }' > /dev/null
        sleep 0.5
    done
    echo "✓ 5 alerts sent! Observe how SRE Copilot deduplicates and groups them into 1 unified incident."
}

test_real_gcp_log() {
    echo ""
    echo "==> Triggering real GCP Cloud Logging alert via gcloud CLI..."
    if ! command -v gcloud &> /dev/null; then
        echo "Error: 'gcloud' CLI is not installed or not in PATH."
        return
    fi
    gcloud logging write sre-app-log \
      "CRITICAL: Payment gateway timed out after 30000ms. Database connection pool exhausted." \
      --severity=ERROR
    echo "✓ Error log written to GCP Cloud Logging! GCP log_error_policy will fire within 30-60 seconds."
}

test_real_gcp_cpu() {
    echo ""
    echo "==> Burning CPU for 130 seconds to trigger real GCP GCE High CPU alert..."
    echo "    (Requires >= 120s of >85% CPU for GCP monitoring policy to fire)"
    python3 -c "import time, multiprocessing; [multiprocessing.Process(target=lambda: [x for x in iter(int, 1)]).start() for _ in range(4)]; time.sleep(130)" || true
    echo "✓ CPU stress completed! Check GCP Alert Policy incident."
}

get_cloudrun_url_and_token() {
    if [ -z "${SAMPLE_SERVICE_URL:-}" ]; then
        read -p "Enter Cloud Run Sample Nginx URL (e.g. https://sample-nginx-service-...run.app): " SAMPLE_SERVICE_URL
        SAMPLE_SERVICE_URL="${SAMPLE_SERVICE_URL%/}"
    fi

    AUTH_HEADER=""
    if command -v gcloud &> /dev/null; then
        TOKEN=$(gcloud auth print-identity-token 2>/dev/null || true)
        if [ -n "$TOKEN" ]; then
            AUTH_HEADER="Authorization: Bearer $TOKEN"
        fi
    fi
}

test_real_gcp_cloudrun_4xx() {
    get_cloudrun_url_and_token
    echo ""
    echo "==> Sending 15 HTTP 404 requests to ${SAMPLE_SERVICE_URL}/trigger-404-error..."
    for i in $(seq 1 15); do
        if [ -n "${AUTH_HEADER}" ]; then
            CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "${AUTH_HEADER}" "${SAMPLE_SERVICE_URL}/trigger-404-error" || true)
        else
            CODE=$(curl -s -o /dev/null -w "%{http_code}" "${SAMPLE_SERVICE_URL}/trigger-404-error" || true)
        fi
        echo "  Request $i: HTTP $CODE"
        sleep 0.5
    done
    echo "✓ 15 HTTP 404 errors sent to Cloud Run! GCP Cloud Run 4xx metric alert policy (>10 errors/min) will fire within 60-120 seconds."
}

test_real_gcp_cloudrun_5xx() {
    get_cloudrun_url_and_token
    echo ""
    echo "==> Sending 10 HTTP 500 requests to ${SAMPLE_SERVICE_URL}/status/500..."
    for i in $(seq 1 10); do
        if [ -n "${AUTH_HEADER}" ]; then
            CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "${AUTH_HEADER}" "${SAMPLE_SERVICE_URL}/status/500" || true)
        else
            CODE=$(curl -s -o /dev/null -w "%{http_code}" "${SAMPLE_SERVICE_URL}/status/500" || true)
        fi
        echo "  Request $i: HTTP $CODE"
        sleep 1
    done
    echo "✓ 10 HTTP 500 errors sent to Cloud Run! GCP Cloud Run 5xx metric alert policy will fire within 60-120 seconds."
}

# Main loop
while true; do
    show_menu
    case "$choice" in
        1) test_monitoring_webhook ;;
        2) test_logging_webhook ;;
        3) test_alert_storm ;;
        4) test_real_gcp_log ;;
        5) test_real_gcp_cpu ;;
        6) test_real_gcp_cloudrun_4xx ;;
        7) test_real_gcp_cloudrun_5xx ;;
        q|Q) echo "Exiting."; exit 0 ;;
        *) echo "Invalid choice. Please select 1-7 or q." ;;
    esac
done
