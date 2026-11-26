output "vm_public_ip" {
  description = "Public IP address of the SRE Copilot Compute Engine VM."
  value       = google_compute_instance.sre_copilot_vm.network_interface[0].access_config[0].nat_ip
}

output "sre_copilot_ui_url" {
  description = "Direct URL to access the SRE Copilot Web Dashboard via Nginx reverse proxy (Port 80)."
  value       = "http://${google_compute_instance.sre_copilot_vm.network_interface[0].access_config[0].nat_ip}"
}

output "gcp_monitoring_webhook_url" {
  description = "Webhook endpoint configured in GCP Cloud Monitoring notification channels (Port 80)."
  value       = "http://${google_compute_instance.sre_copilot_vm.network_interface[0].access_config[0].nat_ip}/api/v1/webhooks/gcp/monitoring"
}

output "gcp_logging_webhook_url" {
  description = "Webhook endpoint for GCP Cloud Logging export sinks (Port 80)."
  value       = "http://${google_compute_instance.sre_copilot_vm.network_interface[0].access_config[0].nat_ip}/api/v1/webhooks/gcp/logging"
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
