output "vm_public_ip" {
  description = "Reserved Static Public IP address of the SRE Copilot Compute Engine VM."
  value       = google_compute_address.vm_static_ip.address
}

output "sre_copilot_ui_url" {
  description = "Direct URL to access the SRE Copilot Web Dashboard via Nginx reverse proxy (Port 80)."
  value       = "http://${google_compute_address.vm_static_ip.address}"
}

output "gcp_monitoring_webhook_url" {
  description = "Webhook endpoint configured in GCP Cloud Monitoring notification channels (Port 80)."
  value       = "http://${google_compute_address.vm_static_ip.address}/api/v1/webhooks/gcp/monitoring"
}

output "gcp_logging_webhook_url" {
  description = "Webhook endpoint for GCP Cloud Logging export sinks (Port 80)."
  value       = "http://${google_compute_address.vm_static_ip.address}/api/v1/webhooks/gcp/logging"
}

output "service_account_email" {
  description = "The Read-Only Service Account attached to the VM for GCP inspection."
  value       = google_service_account.sre_copilot_sa.email
}

output "deployer_service_account_email" {
  description = "The CI/CD Deployer Service Account email for Workload Identity Federation."
  value       = google_service_account.sre_deployer_sa.email
}

output "workload_identity_provider" {
  description = "Workload Identity Provider resource name for GitHub Actions secret (GCP_WORKLOAD_IDENTITY_PROVIDER)."
  value       = "${google_iam_workload_identity_pool.github_pool.name}/providers/${google_iam_workload_identity_pool_provider.github_provider.workload_identity_pool_provider_id}"
}

output "workload_identity_pool_id" {
  description = "The Workload Identity Pool ID."
  value       = google_iam_workload_identity_pool.github_pool.workload_identity_pool_id
}

output "cloud_sql_instance_name" {
  description = "The Cloud SQL PostgreSQL 16 instance name."
  value       = google_sql_database_instance.postgres_instance.name
}

output "cloud_sql_private_ip" {
  description = "Private VPC IP address of the Cloud SQL PostgreSQL instance."
  value       = google_sql_database_instance.postgres_instance.private_ip_address
}

output "artifacts_bucket_name" {
  description = "Cloud Storage bucket name for postmortem and RCA report archives."
  value       = google_storage_bucket.artifacts_bucket.name
}

output "sample_service_url" {
  description = "Public URL of the sample Cloud Run Nginx microservice for metric alert testing."
  value       = try(google_cloud_run_service.sample_test_service[0].status[0].url, "")
}

output "sample_service_test_4xx_cmd" {
  description = "Command to trigger real GCP Cloud Run 4xx client error alert (sends 15x 404 requests)."
  value       = try("for i in {1..15}; do curl -s -o /dev/null -w \"%%{http_code}\\n\" -H \"Authorization: Bearer $(gcloud auth print-identity-token)\" \"${google_cloud_run_service.sample_test_service[0].status[0].url}/trigger-404-error\"; sleep 1; done", "")
}

output "sample_service_test_5xx_cmd" {
  description = "Command to trigger real GCP Cloud Run 5xx server error alert."
  value       = try("for i in {1..10}; do curl -s -o /dev/null -w \"%%{http_code}\\n\" -H \"Authorization: Bearer $(gcloud auth print-identity-token)\" \"${google_cloud_run_service.sample_test_service[0].status[0].url}/trigger-500-error\"; sleep 1; done", "")
}
