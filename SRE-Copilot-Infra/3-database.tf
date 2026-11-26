# ==============================================================================
# 3. Managed PostgreSQL Database (Cloud SQL for PostgreSQL 16) (From YAML)
# ==============================================================================

resource "random_password" "db_password" {
  length  = 20
  special = false
}

locals {
  effective_db_password = try(local.cfg.database.db_password, "") != "" ? local.cfg.database.db_password : random_password.db_password.result
}

resource "google_sql_database_instance" "postgres_instance" {
  name                = "${local.cfg.database.instance_name_prefix}-${random_id.suffix.hex}"
  database_version    = local.cfg.database.version
  region              = local.cfg.project.region
  deletion_protection = false # Set to true for strict production retention

  settings {
    tier = local.cfg.database.tier

    ip_configuration {
      ipv4_enabled    = true
      private_network = google_compute_network.sre_vpc.id
    }

    backup_configuration {
      enabled    = local.cfg.database.backup_enabled
      start_time = local.cfg.database.backup_start_time
    }

    database_flags {
      name  = "max_connections"
      value = tostring(local.cfg.database.max_connections)
    }
  }

  depends_on = [google_service_networking_connection.private_vpc_connection]
}

resource "google_sql_database" "sre_copilot_database" {
  name     = local.cfg.database.db_name
  instance = google_sql_database_instance.postgres_instance.name
}

resource "google_sql_user" "sre_db_user" {
  name     = local.cfg.database.db_user
  instance = google_sql_database_instance.postgres_instance.name
  password = local.effective_db_password
}
