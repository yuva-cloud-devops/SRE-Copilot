# SRE Copilot GCP Infrastructure (YAML-Driven Terraform)

This directory contains production-ready Terraform modules where **100% of infrastructure configuration is driven by a single YAML file (`config.yaml`)**.

---

## 📁 Directory Structure & Numbered Files

```
SRE-Copilot-Infra/
├── config.yaml               # 🌟 Central YAML configuration (Everything configured here!)
├── config.yaml.example       # Template example for different environments
├── 0-provider.tf             # yamldecode("config.yaml"), GCP & Random provider initialization
├── 1-vpc.tf                  # VPC, Subnet, Private Peering & Firewall Rules from YAML
├── 2-iam.tf                  # Read-Only Service Account & Project IAM Roles
├── 3-database.tf             # Managed Cloud SQL PostgreSQL 16 (Private VPC Peering)
├── 4-vm.tf                   # Compute Engine VM running SRE Copilot & startup script
├── 5-storage.tf              # GCS Bucket for RCA postmortems and report archives
├── 6-monitoring.tf           # Part 1: Cloud Monitoring & Logging Alert Policies (Metric & Log Alerts)
├── 7-workload-identity.tf    # Workload Identity Pool & Provider for GitHub Actions OIDC
├── 8-sample-test-service.tf  # Sample Nginx Cloud Run Microservice for Real Metric Alert Testing
├── variables.tf              # Points to config_file (default: config.yaml)
├── outputs.tf                # Exported outputs (VM IP, Webhook URLs, Sample Service URLs)
└── scripts/
    ├── startup.sh            # Automated VM bootstrap script (Systemd service & Nginx)
    └── test-alerts.sh        # Alert Testing Suite (Instant simulation & Real GCP metrics)
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

workload_identity:
  pool_id: "sre-github-pool"
  provider_id: "sre-github-provider"
  github_repository: "yuva-cloud-devops/SRE-Copilot"

monitoring:
  webhook_name: "SRE Copilot Monitoring Webhook"
  logging_webhook_name: "SRE Copilot Logging Webhook"
  cpu_threshold: 0.85
  cpu_duration: "120s"
  cloudsql_cpu_threshold: 0.80
  cloudsql_memory_threshold: 0.85
  cloudrun_4xx_threshold: 10 # Rate of 4xx responses/minute
  cloudrun_5xx_threshold: 5
  cloudrun_latency_threshold_ms: 2000

# Sample Nginx Microservice on Cloud Run for Real GCP Metric Alert Testing
sample_service:
  enabled: true
  name: "sample-nginx-service"
  image: "nginx:alpine"
  container_port: 80
  invoker_members:
    - "user:yuvaraj.k@quadrasystems.net"
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

---

## 🧪 Testing GCP Metric & Log Alerts

Use [`scripts/test-alerts.sh`](scripts/test-alerts.sh) to test all alert scenarios:

```bash
chmod +x scripts/test-alerts.sh

# Run interactive test suite
./scripts/test-alerts.sh http://<VM_IP> <SAMPLE_CLOUDRUN_URL>
```

### Direct Commands to Trigger Real GCP Alerts:

1. **Cloud Run 4xx Client Error Alert (sends 15x 404s)**:
   ```bash
   TOKEN=$(gcloud auth print-identity-token)
   for i in {1..15}; do
     curl -s -o /dev/null -w "%{http_code}\n" \
       -H "Authorization: Bearer $TOKEN" \
       "<SAMPLE_CLOUDRUN_URL>/trigger-404-error"
     sleep 0.5
   done
   ```
2. **Cloud Run 5xx Server Error Alert**:
   ```bash
   TOKEN=$(gcloud auth print-identity-token)
   for i in {1..10}; do
     curl -s -o /dev/null -w "%{http_code}\n" \
       -H "Authorization: Bearer $TOKEN" \
       "<SAMPLE_CLOUDRUN_URL>/trigger-500-error"
     sleep 1
   done
   ```
3. **Cloud Logging ERROR Alert**:
   ```bash
   gcloud logging write sre-app-log "CRITICAL: Database connection pool exhausted" --severity=ERROR
   ```
4. **VM High CPU Saturation Alert**:
   ```bash
   # On the VM:
   python3 -c "import time, multiprocessing; [multiprocessing.Process(target=lambda: [x for x in iter(int, 1)]).start() for _ in range(4)]; time.sleep(130)"
   ```
