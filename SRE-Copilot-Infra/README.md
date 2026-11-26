# SRE Copilot GCP Infrastructure (YAML-Driven Terraform)

This directory contains production-ready Terraform modules where **100% of infrastructure configuration is driven by a single YAML file (`config.yaml`)**.

---

## 📁 Directory Structure & Numbered Files

```
SRE-Copilot-Infra/
├── config.yaml             # 🌟 Central YAML configuration (Everything configured here!)
├── config.yaml.example     # Template example for different environments
├── 0-provider.tf           # yamldecode("config.yaml"), GCP & Random provider initialization
├── 1-vpc.tf                # VPC, Subnet, Private Peering & Firewall Rules from YAML
├── 2-iam.tf                # Read-Only Service Account & Project IAM Roles
├── 3-database.tf           # Managed Cloud SQL PostgreSQL 16 (Private VPC Peering)
├── 4-vm.tf                 # Compute Engine VM running SRE Copilot & startup script
├── 5-storage.tf            # GCS Bucket for RCA postmortems and report archives
├── 6-monitoring.tf         # Part 1: Cloud Monitoring & Logging Alert Policies (Metric & Log Alerts)
├── 7-workload-identity.tf  # Workload Identity Pool & Provider for GitHub Actions OIDC
├── variables.tf            # Points to config_file (default: config.yaml)
├── outputs.tf              # Exported outputs (VM IP, Webhook URLs, DB details)
└── scripts/
    └── startup.sh          # Automated VM bootstrap script (Systemd service)
```

---

## ⚙️ Central YAML Configuration (`config.yaml`)

Edit `config.yaml` to customize your GCP project and infrastructure:

```yaml
project:
  id: "your-gcp-project-id"
  region: "us-central1"
  zone: "us-central1-a"
  environment: "production"

network:
  vpc_name: "sre-copilot-vpc"
  subnet_name: "sre-copilot-subnet"
  subnet_cidr: "10.10.0.0/24"
  http_ports: ["8080", "80", "443"]
  ssh_ports: ["22"]
  iap_cidr: "35.235.240.0/20"

database:
  instance_name_prefix: "sre-copilot-db"
  version: "POSTGRES_16"
  tier: "db-f1-micro"
  db_name: "sre_copilot"
  db_user: "sre_admin"
  db_password: "" # Leave empty to auto-generate a secure random 20-character password
  max_connections: 100
  backup_enabled: true
  backup_start_time: "02:00"

vm:
  name: "sre-copilot-server"
  machine_type: "e2-medium"
  os_image: "debian-cloud/debian-12"
  disk_size_gb: 30
  disk_type: "pd-balanced"

storage:
  bucket_prefix: "sre-copilot-artifacts"
  retention_days: 365
  versioning_enabled: true

monitoring:
  webhook_name: "SRE Copilot Ingestion Webhook"
  alert_policy_name: "Critical: Compute Engine / Cloud Run High Saturation"
  cpu_threshold: 0.85
  cpu_duration: "120s"
```

---

## 🚀 Quickstart: Provisioning

```bash
cd SRE-Copilot-Infra

# 1. Update your GCP Project ID in config.yaml
# (Open config.yaml and change project.id to your project)

# 2. Initialize and deploy
terraform init
terraform plan
terraform apply
```

To use an alternate YAML file (e.g. `staging.yaml`):
```bash
terraform apply -var="config_file=staging.yaml"
```
