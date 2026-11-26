# ==============================================================================
# 4. Compute Engine Virtual Machine (From YAML)
# ==============================================================================

# Reserve a static external public IP for permanent UI & webhook access
resource "google_compute_address" "vm_static_ip" {
  name   = "${local.cfg.vm.name}-static-ip"
  region = local.cfg.project.region
}

resource "google_compute_instance" "sre_copilot_vm" {
  name         = local.cfg.vm.name
  machine_type = local.cfg.vm.machine_type
  zone         = local.cfg.project.zone
  tags         = ["sre-copilot-server"]

  boot_disk {
    initialize_params {
      image = local.cfg.vm.os_image
      size  = local.cfg.vm.disk_size_gb
      type  = local.cfg.vm.disk_type
    }
  }

  network_interface {
    network    = google_compute_network.sre_vpc.id
    subnetwork = google_compute_subnetwork.sre_subnet.id

    access_config {
      nat_ip = google_compute_address.vm_static_ip.address
    }
  }

  service_account {
    email  = google_service_account.sre_copilot_sa.email
    scopes = ["cloud-platform"]
  }

  metadata = {
    enable-oslogin = "TRUE"
    startup-script = templatefile("${path.module}/scripts/startup.sh", {
      PROJECT_ID     = local.cfg.project.id
      DEFAULT_REGION = local.cfg.project.region
      DB_HOST        = google_sql_database_instance.postgres_instance.private_ip_address
      DB_USER        = local.cfg.database.db_user
      DB_PASSWORD    = local.effective_db_password
      DB_NAME        = local.cfg.database.db_name
      USER_NAME      = "srecopilot"
      STORAGE_BUCKET = google_storage_bucket.artifacts_bucket.name
    })
  }

  depends_on = [
    google_sql_database_instance.postgres_instance,
    google_sql_user.sre_db_user
  ]
}
