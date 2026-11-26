# ==============================================================================
# 6. Part 1: GCP Cloud Monitoring & Cloud Logging Alert Policies (From YAML)
# Wires Metric Alerts and Error Log Alerts directly to SRE Copilot Ingestion Webhooks
# ==============================================================================

# ------------------------------------------------------------------------------
# 1. Webhook Notification Channels pointing to SRE Copilot VM via Nginx (Port 80)
# ------------------------------------------------------------------------------

# Channel 1: Cloud Monitoring Metric Alert Webhook
resource "google_monitoring_notification_channel" "sre_copilot_webhook" {
  display_name = local.cfg.monitoring.webhook_name
  type         = "webhook_tokenauth"

  labels = {
    url = "http://${google_compute_address.vm_static_ip.address}/api/v1/webhooks/gcp/monitoring"
  }

  user_labels = {
    managed_by = "terraform"
    target     = "sre-copilot-metric-alerts"
  }
}

# Channel 2: Cloud Logging Error Log Webhook
resource "google_monitoring_notification_channel" "sre_copilot_logging_webhook" {
  display_name = try(local.cfg.monitoring.logging_webhook_name, "SRE Copilot Logging Webhook")
  type         = "webhook_tokenauth"

  labels = {
    url = "http://${google_compute_address.vm_static_ip.address}/api/v1/webhooks/gcp/logging"
  }

  user_labels = {
    managed_by = "terraform"
    target     = "sre-copilot-log-alerts"
  }
}

# ------------------------------------------------------------------------------
# 2. Metric Alerts: Compute Engine (GCE)
# ------------------------------------------------------------------------------

# Alert 1: GCE VM High CPU Saturation
resource "google_monitoring_alert_policy" "vm_high_cpu" {
  display_name = "Critical: Compute Engine High CPU Saturation (> ${local.cfg.monitoring.cpu_threshold * 100}%)"
  combiner     = "OR"

  conditions {
    display_name = "GCE Instance CPU Utilization > ${local.cfg.monitoring.cpu_threshold * 100}%"
    condition_threshold {
      filter          = "resource.type = \"gce_instance\" AND metric.type = \"compute.googleapis.com/instance/cpu/utilization\""
      comparison      = "COMPARISON_GT"
      threshold_value = local.cfg.monitoring.cpu_threshold
      duration        = local.cfg.monitoring.cpu_duration

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.sre_copilot_webhook.name]
}

# ------------------------------------------------------------------------------
# 3. Metric Alerts: Cloud SQL (PostgreSQL Database)
# ------------------------------------------------------------------------------

# Alert 2: Cloud SQL High CPU Utilization
resource "google_monitoring_alert_policy" "cloudsql_high_cpu" {
  display_name = "Critical: Cloud SQL High CPU Saturation (> ${try(local.cfg.monitoring.cloudsql_cpu_threshold, 0.80) * 100}%)"
  combiner     = "OR"

  conditions {
    display_name = "Cloud SQL CPU Utilization > ${try(local.cfg.monitoring.cloudsql_cpu_threshold, 0.80) * 100}%"
    condition_threshold {
      filter          = "resource.type = \"cloudsql_database\" AND metric.type = \"cloudsql.googleapis.com/database/cpu/utilization\""
      comparison      = "COMPARISON_GT"
      threshold_value = try(local.cfg.monitoring.cloudsql_cpu_threshold, 0.80)
      duration        = "120s"

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.sre_copilot_webhook.name]
}

# Alert 3: Cloud SQL High Memory Utilization
resource "google_monitoring_alert_policy" "cloudsql_high_memory" {
  display_name = "Critical: Cloud SQL High Memory Saturation (> ${try(local.cfg.monitoring.cloudsql_memory_threshold, 0.85) * 100}%)"
  combiner     = "OR"

  conditions {
    display_name = "Cloud SQL Memory Utilization > ${try(local.cfg.monitoring.cloudsql_memory_threshold, 0.85) * 100}%"
    condition_threshold {
      filter          = "resource.type = \"cloudsql_database\" AND metric.type = \"cloudsql.googleapis.com/database/memory/utilization\""
      comparison      = "COMPARISON_GT"
      threshold_value = try(local.cfg.monitoring.cloudsql_memory_threshold, 0.85)
      duration        = "120s"

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.sre_copilot_webhook.name]
}

# ------------------------------------------------------------------------------
# 4. Metric Alerts: Cloud Run (Microservices / API Traffic)
# ------------------------------------------------------------------------------

# Alert 4a: Cloud Run 4xx Client Error Rate Spike
resource "google_monitoring_alert_policy" "cloudrun_4xx_errors" {
  display_name = "Warning: Cloud Run 4xx Client Error Rate Spike"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Run 4xx Responses Count > ${try(local.cfg.monitoring.cloudrun_4xx_threshold, 10)}"
    condition_threshold {
      filter          = "resource.type = \"cloud_run_revision\" AND metric.type = \"run.googleapis.com/request_count\" AND metric.labels.response_code_class = \"4xx\""
      comparison      = "COMPARISON_GT"
      threshold_value = try(local.cfg.monitoring.cloudrun_4xx_threshold, 10)
      duration        = "60s"

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_RATE"
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.sre_copilot_webhook.name]
}

# Alert 4b: Cloud Run 5xx Server Error Rate Spike
resource "google_monitoring_alert_policy" "cloudrun_5xx_errors" {
  display_name = "Critical: Cloud Run 5xx Server Error Rate Spike"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Run 5xx Responses Count > ${try(local.cfg.monitoring.cloudrun_5xx_threshold, 5)}"
    condition_threshold {
      filter          = "resource.type = \"cloud_run_revision\" AND metric.type = \"run.googleapis.com/request_count\" AND metric.labels.response_code_class = \"5xx\""
      comparison      = "COMPARISON_GT"
      threshold_value = try(local.cfg.monitoring.cloudrun_5xx_threshold, 5)
      duration        = "60s"

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_RATE"
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.sre_copilot_webhook.name]
}

# Alert 5: Cloud Run High Request Latency (p95 > Threshold)
resource "google_monitoring_alert_policy" "cloudrun_high_latency" {
  display_name = "Warning: Cloud Run Request Latency Degradation (p95 > ${try(local.cfg.monitoring.cloudrun_latency_threshold_ms, 2000)}ms)"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Run Latency p95 > ${try(local.cfg.monitoring.cloudrun_latency_threshold_ms, 2000)}ms"
    condition_threshold {
      filter          = "resource.type = \"cloud_run_revision\" AND metric.type = \"run.googleapis.com/request_latencies\""
      comparison      = "COMPARISON_GT"
      threshold_value = try(local.cfg.monitoring.cloudrun_latency_threshold_ms, 2000)
      duration        = "120s"

      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_PERCENTILE_95"
        cross_series_reducer = "REDUCE_MEAN"
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.sre_copilot_webhook.name]
}

# ------------------------------------------------------------------------------
# 5. Log-Based Alert: Critical Error Logs Across Services (Severity >= ERROR)
# ------------------------------------------------------------------------------

# Alert 6: Cloud Logging Error Log Alerts (Severity >= ERROR)
resource "google_monitoring_alert_policy" "log_error_policy" {
  display_name = "Critical: Infrastructure & Application Error Logs (Severity >= ERROR)"
  combiner     = "OR"

  conditions {
    display_name = "Critical or Error Log Entry Detected"
    condition_matched_log {
      filter = "severity >= ERROR"
    }
  }

  alert_strategy {
    notification_rate_limit {
      period = "300s" # Throttles alert bursts during an outage to prevent alert storms
    }
  }

  notification_channels = [google_monitoring_notification_channel.sre_copilot_webhook.name]
}
