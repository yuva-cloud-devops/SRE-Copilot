# SRE Copilot Platform

AI-Powered Incident Intelligence, Alert Correlation, and Automated Postmortem Engine for SRE & DevOps teams.

---

## 📁 Repository Structure

```
SRE-Copilot/
├── .github/
│   └── workflows/
│       └── ci-cd.yml              # CI/CD Pipeline (Triggers ONLY on changes in SRE-Copilot-Code/**)
│
├── SRE-Copilot-Code/              # Application Source Code
│   ├── cmd/server/main.go         # Go Monolith entry point
│   ├── config.json                # Local development runtime configuration
│   ├── internal/
│   │   ├── config/                # Environment & JSON config parser
│   │   ├── database/              # PostgreSQL schema, migrations & pooling
│   │   ├── handlers/              # Web views, REST APIs & GCP Webhook receivers
│   │   ├── models/                # Incidents, Alerts, GCP Logs & RCA models
│   │   ├── repository/            # PostgreSQL & In-Memory fallback repositories
│   │   └── services/              # 15-min Sliding Window Correlator, AI Copilot, SSE Broadcaster
│   ├── templates/                 # Server-rendered HTML templates (Go html/template)
│   ├── static/                    # Tailwind CSS & static assets
│   └── docs/                      # PRD, High-Level Design (HLD) & Database/API Spec
│
└── SRE-Copilot-Infra/             # Cloud Infrastructure (GCP Terraform driven by config.yaml)
    ├── config.yaml                # 🌟 Central YAML configuration for all GCP resources
    ├── 0-provider.tf              # yamldecode("config.yaml"), GCP & Random providers
    ├── 1-vpc.tf                   # Dedicated VPC, subnet, private peering & firewall rules
    ├── 2-iam.tf                   # Read-Only Service Account for AI infrastructure inspection
    ├── 3-database.tf              # Managed Cloud SQL PostgreSQL 16 (Private VPC Peering)
    ├── 4-vm.tf                    # Compute Engine VM running SRE Copilot
    ├── 5-storage.tf               # GCS Bucket for RCA postmortems & report archives
    ├── 6-monitoring.tf            # Part 1: GCP Cloud Monitoring & Logging Alert Policies (Metric & Log Alerts)
    ├── 7-workload-identity.tf     # Workload Identity Pool & Provider for GitHub Actions OIDC
    ├── variables.tf / outputs.tf  # Input variables & exported endpoints
    └── scripts/startup.sh         # VM bootstrap (Nginx reverse proxy, build-bin runtime directory)
```

---

## 🚀 CI/CD & Production VM Architecture

1. **GitHub Actions Pipeline (`.github/workflows/ci-cd.yml`)**:
   - Located at repository root, triggered strictly on changes in `SRE-Copilot-Code/**`.
   - **Build & Test Stage**:
     - Runs code quality checks (`go vet ./...` & `go mod verify`).
     - Cross-compiles the pure Go binary (`GOOS=linux GOARCH=amd64`).
     - Packages the complete `build-bin` bundle containing: `sre-copilot` binary, `config.json`, `templates/`, and `static/`.
     - Uploads the complete bundle as a single workflow artifact (`build-bin-bundle`).
   - **Deploy Stage (Workload Identity & IAP)**:
     - Authenticates to Google Cloud via **Workload Identity Federation** (`google-github-actions/auth@v2`) without storing long-lived service account keys.
     - Tunnels securely through **Google Cloud Identity-Aware Proxy (IAP)** using `gcloud compute scp --tunnel-through-iap`.
     - Synchronizes files into the server's runtime directory (`/home/srecopilot/build-bin/`).
     - Restarts all production services (`sudo systemctl restart sre-copilot nginx`) non-interactively.

2. **VM Runtime Structure (`/home/srecopilot/build-bin`)**:
   - No `codebase` folder and no symlinks needed on the VM.
   - The runtime directory `/home/srecopilot/build-bin` contains:
     - `sre-copilot` (the compiled executable binary)
     - `config.json` (auto-generated during VM bootstrap with private Cloud SQL IP)
     - `templates/` (synced by CI/CD)
     - `static/` (synced by CI/CD)

3. **Nginx Reverse Proxy**:
   - Listens on public Port 80 and reverse proxies requests to `http://127.0.0.1:8080`.
   - Configured with SSE streaming support (`proxy_buffering off`, infinite keep-alive timeouts) for real-time AI incident triage tokens and live alert timelines.
   - Logs requests to `/var/log/nginx/sre-copilot-access.log` and `/var/log/nginx/sre-copilot-error.log`.

4. **Application Log Storage & Rotation**:
   - **Live Application Logs**: Persistent on disk at `/var/log/sre-copilot/sre-copilot.log` (captures all standard output, error, and Gin web logs).
   - **Log Rotation (`logrotate`)**: Auto-configured in `/etc/logrotate.d/sre-copilot` with daily rotation, 14-day retention, gzip compression, and `copytruncate` to avoid service interruption.
   - **Real-Time Inspection**: SRE engineers can tail logs at any time via:
     ```bash
     tail -f /var/log/sre-copilot/sre-copilot.log
     ```

---

## 🔐 GitHub Actions Repository Secrets

Configure the following secrets in your GitHub repository (`Settings -> Secrets and variables -> Actions`):

| Secret Name | Description | Example / Source |
| :--- | :--- | :--- |
| `GCP_PROJECT_ID` | Google Cloud Project ID | `your-gcp-project-id` |
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | Workload Identity Pool Provider | `projects/<NUM>/locations/global/workloadIdentityPools/<POOL>/providers/<PROV>` |
| `GCP_SERVICE_ACCOUNT` | CI/CD Deployer Service Account Email | Exported from Terraform output: `deployer_service_account_email` |
| `GEMINI_API_KEY` | Google Gemini API Key | Provided via external team / Google AI Studio |
| `GCP_VM_NAME` *(optional)* | Compute Engine VM Name | `sre-copilot-server` (default) |
| `GCP_ZONE` *(optional)* | GCP Zone | `us-central1-a` (default) |

---

## ⚡ Quickstart

### 1. Run the Application Locally
```bash
cd SRE-Copilot-Code
go run ./cmd/server
```
Visit `http://localhost:8080`.

### 2. Deploy Infrastructure to Google Cloud
```bash
cd SRE-Copilot-Infra
# Update config.yaml with your GCP project ID
terraform init
terraform plan
terraform apply
```
