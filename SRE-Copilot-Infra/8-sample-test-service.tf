# ==============================================================================
# 8. Sample Nginx Microservice on Cloud Run (For Real GCP Metric Alert Testing)
# ==============================================================================
# Deploys a lightweight Nginx container ("sample-nginx-service") to test real GCP metrics:
#   - GET /                    -> 200 OK (Standard Nginx welcome page)
#   - GET /trigger-404-error   -> 404 Not Found (triggers Cloud Run 4xx alert policy)
# ==============================================================================

locals {
  sample_service_enabled  = try(local.cfg.sample_service.enabled, true)
  sample_service_invokers = local.sample_service_enabled ? toset(try(local.cfg.sample_service.invoker_members, ["user:yuvaraj.k@quadrasystems.net"])) : toset([])
}

# Ensure Cloud Run API is enabled
resource "google_project_service" "run_api" {
  count                      = local.sample_service_enabled ? 1 : 0
  service                    = "run.googleapis.com"
  disable_on_destroy         = false
  disable_dependent_services = false
}

resource "google_cloud_run_service" "sample_test_service" {
  count    = local.sample_service_enabled ? 1 : 0
  name     = "${try(local.cfg.sample_service.name, "sample-nginx-service")}-${random_id.suffix.hex}"
  location = local.cfg.project.region

  template {
    spec {
      containers {
        image = try(local.cfg.sample_service.image, "nginx:alpine")
        ports {
          name           = "http1"
          container_port = try(local.cfg.sample_service.container_port, 80)
        }
        resources {
          limits = {
            memory = "256Mi"
            cpu    = "1000m"
          }
        }
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }

  depends_on = [google_project_service.run_api]
}

# Allow authorized members to invoke the service (Compliant with Org Policy Domain Restrictions)
resource "google_cloud_run_service_iam_member" "sample_service_invokers" {
  for_each = local.sample_service_invokers
  service  = google_cloud_run_service.sample_test_service[0].name
  location = google_cloud_run_service.sample_test_service[0].location
  role     = "roles/run.invoker"
  member   = each.value
}
