# ==============================================================================
# 1. VPC Network, Subnet & Firewall Configuration (From YAML)
# ==============================================================================

resource "google_compute_network" "sre_vpc" {
  name                    = local.cfg.network.vpc_name
  auto_create_subnetworks = false
  description             = "Dedicated VPC for SRE Copilot platform and monitoring"
}

resource "google_compute_subnetwork" "sre_subnet" {
  name          = "${local.cfg.network.subnet_name}-${local.cfg.project.region}"
  ip_cidr_range = local.cfg.network.subnet_cidr
  region        = local.cfg.project.region
  network       = google_compute_network.sre_vpc.id
}

# Reserved internal IP range for Cloud SQL Private Services Access (VPC Peering)
resource "google_compute_global_address" "private_ip_address" {
  name          = "${local.cfg.network.vpc_name}-private-ip"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.sre_vpc.id
}

resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = google_compute_network.sre_vpc.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address.name]
}

# Firewall: Allow HTTP traffic to SRE Copilot UI and Ingestion Webhooks
resource "google_compute_firewall" "allow_sre_copilot_http" {
  name    = "allow-${local.cfg.network.vpc_name}-http"
  network = google_compute_network.sre_vpc.name

  allow {
    protocol = "tcp"
    ports    = local.cfg.network.http_ports
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["sre-copilot-server"]
  description   = "Allow incoming alerts from GCP Webhooks and SRE engineer browser access"
}

# Firewall: Allow SSH for administrators (and Google IAP)
resource "google_compute_firewall" "allow_ssh" {
  name    = "allow-${local.cfg.network.vpc_name}-ssh"
  network = google_compute_network.sre_vpc.name

  allow {
    protocol = "tcp"
    ports    = local.cfg.network.ssh_ports
  }

  # Allow Google Cloud Identity-Aware Proxy (IAP) CIDR for secure bastionless SSH
  source_ranges = [local.cfg.network.iap_cidr, "0.0.0.0/0"]
  target_tags   = ["sre-copilot-server"]
}

# Firewall: Allow PostgreSQL traffic on port 5432
resource "google_compute_firewall" "allow_postgres" {
  name    = "allow-${local.cfg.network.vpc_name}-postgres"
  network = google_compute_network.sre_vpc.name

  allow {
    protocol = "tcp"
    ports    = ["5432"]
  }

  source_ranges = ["0.0.0.0/0"]
  description   = "Allow PostgreSQL database access on port 5432 from 0.0.0.0/0"
}
