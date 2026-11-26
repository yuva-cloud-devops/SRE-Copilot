# ==============================================================================
# 2. Service Account & IAM Roles for Read-Only AI Resource Inspection
# ==============================================================================

resource "google_service_account" "sre_copilot_sa" {
  account_id   = "sre-copilot-sa"
  display_name = "SRE Copilot Read-Only Inspector & App Runner"
  description  = "Attached to VM to provide read-only inspection of Cloud Run, GCE, Vertex AI, and Cloud SQL"
}

# Grant Read-Only Viewer roles for deep infrastructure inspection
locals {
  copilot_roles = [
    "roles/viewer",                  # Broad read-only inspection of project resources
    "roles/run.viewer",              # Cloud Run service specs, memory/CPU limits, env vars
    "roles/compute.viewer",          # Compute Engine VM machine types, disk status, network tags
    "roles/aiplatform.viewer",       # Vertex AI / Gemini Enterprise endpoints, quotas, and logs
    "roles/storage.objectViewer",    # Cloud Storage bucket configuration and object reading
    "roles/monitoring.viewer",       # Read Cloud Monitoring metric data & alerting policies
    "roles/logging.viewer",          # Read Cloud Logging error logs and log sinks
    "roles/cloudsql.client"          # Secure connection to Cloud SQL PostgreSQL instance
  ]

  deployer_roles = [
    "roles/iap.tunnelResourceAccessor", # Tunnel through IAP for SSH and SCP file transfer
    "roles/compute.instanceAdmin.v1",    # Manage Compute Engine VM for deployment
    "roles/compute.osAdminLogin",        # OS Login administrator rights for SSH commands on VM
    "roles/iam.serviceAccountUser"       # Service Account user
  ]
}

resource "google_project_iam_member" "copilot_iam_bindings" {
  for_each = toset(locals.copilot_roles)
  project  = local.cfg.project.id
  role     = each.key
  member   = "serviceAccount:${google_service_account.sre_copilot_sa.email}"
}

# Dedicated Service Account for GitHub Actions CI/CD Deployment via Workload Identity & IAP
resource "google_service_account" "sre_deployer_sa" {
  account_id   = "sre-deployer-sa"
  display_name = "SRE Copilot CI/CD Deployer"
  description  = "Used by GitHub Actions with Workload Identity Federation to deploy via IAP"
}

resource "google_project_iam_member" "deployer_iam_bindings" {
  for_each = toset(locals.deployer_roles)
  project  = local.cfg.project.id
  role     = each.key
  member   = "serviceAccount:${google_service_account.sre_deployer_sa.email}"
}
