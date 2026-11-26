# ==============================================================================
# 5. Cloud Storage Bucket (From YAML)
# ==============================================================================

resource "google_storage_bucket" "artifacts_bucket" {
  name                        = "${local.cfg.storage.bucket_prefix}-${local.cfg.project.id}-${random_id.suffix.hex}"
  location                    = local.cfg.project.region
  force_destroy               = false
  uniform_bucket_level_access = true

  versioning {
    enabled = local.cfg.storage.versioning_enabled
  }

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      age = local.cfg.storage.retention_days
    }
  }
}
