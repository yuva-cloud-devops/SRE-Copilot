# ==============================================================================
# 7. Workload Identity Federation for GitHub Actions OIDC (From YAML)
# ==============================================================================

# 1. Create the Workload Identity Pool
resource "google_iam_workload_identity_pool" "github_pool" {
  workload_identity_pool_id = local.cfg.workload_identity.pool_id
  display_name              = "SRE Copilot GitHub Actions Pool"
  description               = "Identity pool for GitHub Actions OIDC authentication"
  disabled                  = false
}

# 2. Create the OIDC Workload Identity Provider for GitHub Actions
resource "google_iam_workload_identity_pool_provider" "github_provider" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github_pool.workload_identity_pool_id
  workload_identity_pool_provider_id = local.cfg.workload_identity.provider_id
  display_name                       = "GitHub Actions OIDC Provider"
  description                        = "Allows GitHub Actions runner to authenticate securely without stored keys"
  disabled                           = false

  attribute_mapping = {
    "google.subject"             = "assertion.sub"
    "attribute.actor"            = "assertion.actor"
    "attribute.repository"       = "assertion.repository"
    "attribute.repository_owner" = "assertion.repository_owner"
  }

  attribute_condition = local.cfg.workload_identity.github_repository == "*" ? null : "assertion.repository == '${local.cfg.workload_identity.github_repository}'"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

# 3. Allow GitHub Actions to impersonate the CI/CD Deployer Service Account
resource "google_service_account_iam_member" "deployer_workload_identity" {
  service_account_id = google_service_account.sre_deployer_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = local.cfg.workload_identity.github_repository == "*" ? "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github_pool.name}/*" : "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github_pool.name}/attribute.repository/${local.cfg.workload_identity.github_repository}"
}
