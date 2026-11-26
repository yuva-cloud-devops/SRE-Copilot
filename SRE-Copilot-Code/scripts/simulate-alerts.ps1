# ==============================================================================
# SRE Copilot - End-to-End Alert Simulation Script (PowerShell)
# Sends realistic Google Cloud Monitoring and Cloud Logging payloads to the SRE Copilot webhook
# ==============================================================================

param (
    [string]$TargetHost = "http://34.47.244.171"
)

Write-Host "Targeting SRE Copilot at: $TargetHost" -ForegroundColor Cyan

$monitoringUrl = "$TargetHost/api/v1/webhooks/gcp/monitoring"
$loggingUrl = "$TargetHost/api/v1/webhooks/gcp/logging"
$epochNow = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$isoNow = [DateTime]::UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ")

Write-Host "`n1. Simulating GCP Cloud Monitoring Alert: Cloud Run 5xx Spike..." -ForegroundColor Yellow
$alert5xx = @{
    version = "1.2"
    incident = @{
        incident_id = "0.m89abc123def456-5xx"
        resource_id = "sample-nginx-service-dfe66177"
        resource_name = "sample-nginx-service-dfe66177"
        resource = @{
            type = "cloud_run_revision"
            labels = @{
                service_name = "sample-nginx-service"
                project_id = "yuvaraj-internal-project"
                location = "asia-south1"
            }
        }
        policy_name = "Cloud Run 5xx Server Errors Detected"
        condition_name = "Cloud Run request count with response_code_class = 5xx > 5 req/min"
        url = "https://console.cloud.google.com/monitoring/alerting/incidents/0.m89abc123def456"
        state = "open"
        started_at = $epochNow
        summary = "Cloud Run service sample-nginx-service is experiencing sustained HTTP 500 internal server errors."
    }
} | ConvertTo-Json -Depth 5

Invoke-RestMethod -Uri $monitoringUrl -Method Post -Body $alert5xx -ContentType "application/json" | Format-List

Start-Sleep -Seconds 2

Write-Host "`n2. Simulating GCP Cloud Monitoring Alert: Cloud Run Latency Degradation..." -ForegroundColor Yellow
$alertLatency = @{
    version = "1.2"
    incident = @{
        incident_id = "0.m89abc123def456-latency"
        resource_id = "sample-nginx-service-dfe66177"
        resource_name = "sample-nginx-service-dfe66177"
        resource = @{
            type = "cloud_run_revision"
            labels = @{
                service_name = "sample-nginx-service"
                project_id = "yuvaraj-internal-project"
                location = "asia-south1"
            }
        }
        policy_name = "Cloud Run p95 Latency Degradation (>2000ms)"
        condition_name = "95th percentile latency > 2s over 2 minutes"
        url = "https://console.cloud.google.com/monitoring/alerting/incidents/0.m89abc123def456-latency"
        state = "open"
        started_at = $epochNow
        summary = "Cloud Run service sample-nginx-service p95 request latency degraded to 3420ms."
    }
} | ConvertTo-Json -Depth 5

Invoke-RestMethod -Uri $monitoringUrl -Method Post -Body $alertLatency -ContentType "application/json" | Format-List

Start-Sleep -Seconds 2

Write-Host "`n3. Simulating Cloud Logging Log Sink: Fatal DB Error..." -ForegroundColor Yellow
$logEntry = @{
    insertId = "log-entry-uuid-998877"
    severity = "CRITICAL"
    textPayload = "FATAL: remaining connection slots are reserved for non-replication superuser connections. Total active pool connections: 100/100."
    resource = @{
        type = "cloudsql_database"
        labels = @{
            database_id = "yuvaraj-internal-project:sre-copilot-db-dfe66177"
            project_id = "yuvaraj-internal-project"
            region = "asia-south1"
        }
    }
    timestamp = $isoNow
    labels = @{
        service = "sample-nginx-service"
        environment = "production"
    }
} | ConvertTo-Json -Depth 5

Invoke-RestMethod -Uri $loggingUrl -Method Post -Body $logEntry -ContentType "application/json" | Format-List

Write-Host "`n✓ Simulation Complete! View live correlated incident on Dashboard: $TargetHost" -ForegroundColor Green
