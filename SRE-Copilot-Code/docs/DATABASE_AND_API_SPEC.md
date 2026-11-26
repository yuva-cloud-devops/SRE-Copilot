# Database Schema & Route / API Specification
## SRE Copilot: Data Models, UI Template Routes & Webhook Contracts

- **Document Version:** 1.1.0
- **Status:** Approved / Architecture Ready
- **Architecture:** Go Monolith with `html/template` & REST/SSE Endpoints
- **Database Engine:** PostgreSQL 16+ with `pgvector`

---

# Part 1: PostgreSQL + pgvector Database Schema

```sql
-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Enum Types
CREATE TYPE user_role AS ENUM ('admin', 'responder', 'viewer');
CREATE TYPE incident_severity AS ENUM ('P1', 'P2', 'P3', 'P4');
CREATE TYPE incident_status AS ENUM ('investigating', 'identified', 'monitoring', 'resolved');
CREATE TYPE alert_status AS ENUM ('firing', 'resolved', 'suppressed');
CREATE TYPE event_type AS ENUM ('alert_triggered', 'status_changed', 'severity_changed', 'comment_added', 'runbook_executed', 'ai_insight');

-- 1. Users Table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'responder',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Services Table (Application Catalog)
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    tier VARCHAR(10) NOT NULL DEFAULT 'tier-1', -- tier-1, tier-2, tier-3
    repository_url VARCHAR(255),
    team_owner VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Incidents Table
CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_number SERIAL UNIQUE,
    title VARCHAR(255) NOT NULL,
    summary TEXT,
    service_id UUID REFERENCES services(id) ON DELETE SET NULL,
    environment VARCHAR(50) NOT NULL DEFAULT 'production',
    severity incident_severity NOT NULL DEFAULT 'P3',
    status incident_status NOT NULL DEFAULT 'investigating',
    lead_responder_id UUID REFERENCES users(id) ON DELETE SET NULL,
    root_cause_hypothesis TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_incidents_status_severity ON incidents(status, severity);
CREATE INDEX idx_incidents_service ON incidents(service_id);
CREATE INDEX idx_incidents_started_at ON incidents(started_at DESC);

-- 4. Alerts Table (Ingested from GCP Cloud Monitoring)
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID REFERENCES incidents(id) ON DELETE CASCADE,
    fingerprint VARCHAR(64) NOT NULL,
    alert_name VARCHAR(255) NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'gcp_monitoring',
    service_name VARCHAR(100) NOT NULL,
    environment VARCHAR(50) NOT NULL DEFAULT 'production',
    severity incident_severity NOT NULL DEFAULT 'P3',
    status alert_status NOT NULL DEFAULT 'firing',
    description TEXT,
    gcp_policy_name VARCHAR(255),
    gcp_condition VARCHAR(255),
    gcp_metric_type VARCHAR(255),
    gcp_console_url TEXT,
    labels JSONB DEFAULT '{}'::jsonb,
    annotations JSONB DEFAULT '{}'::jsonb,
    recurrence_count INT NOT NULL DEFAULT 1,
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_alerts_fingerprint ON alerts(fingerprint);
CREATE INDEX idx_alerts_incident_id ON alerts(incident_id);

-- 5. GCP Logs Table (Ingested from GCP Cloud Logging Export Sinks)
CREATE TABLE gcp_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID REFERENCES incidents(id) ON DELETE CASCADE,
    log_name VARCHAR(255) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    service_name VARCHAR(100) NOT NULL,
    message TEXT NOT NULL,
    payload JSONB DEFAULT '{}'::jsonb,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_gcp_logs_incident ON gcp_logs(incident_id);
CREATE INDEX idx_gcp_logs_timestamp ON gcp_logs(timestamp DESC);
    incident_id UUID REFERENCES incidents(id) ON DELETE SET NULL,
    fingerprint VARCHAR(64) NOT NULL, -- MD5/SHA256 of (service_name + alert_name + environment)
    alert_name VARCHAR(255) NOT NULL,
    source VARCHAR(50) NOT NULL, -- 'prometheus', 'grafana', 'datadog', 'manual'
    service_id UUID REFERENCES services(id) ON DELETE SET NULL,
    environment VARCHAR(50) NOT NULL DEFAULT 'production',
    severity incident_severity NOT NULL DEFAULT 'P3',
    status alert_status NOT NULL DEFAULT 'firing',
    description TEXT,
    labels JSONB NOT NULL DEFAULT '{}',
    annotations JSONB NOT NULL DEFAULT '{}',
    recurrence_count INT NOT NULL DEFAULT 1,
    triggered_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_alerts_fingerprint ON alerts(fingerprint);
CREATE INDEX idx_alerts_incident_id ON alerts(incident_id);
CREATE INDEX idx_alerts_labels ON alerts USING GIN (labels);

-- 5. Incident Timeline Table
CREATE TABLE incident_timeline (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    event_type event_type NOT NULL,
    description TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_timeline_incident ON incident_timeline(incident_id, created_at ASC);

-- 6. Knowledge Documents & Runbooks (Vector Search)
CREATE TABLE knowledge_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    document_type VARCHAR(50) NOT NULL, -- 'runbook', 'historical_rca', 'architecture'
    service_id UUID REFERENCES services(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    embedding vector(1536), -- Vector dimensions for text-embedding-3-small or Gemini embeddings (768/1536)
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_knowledge_embedding ON knowledge_documents USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_knowledge_service ON knowledge_documents(service_id);

-- 7. RCA Reports Table
CREATE TABLE rca_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID UNIQUE NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    executive_summary TEXT NOT NULL,
    impact_summary TEXT NOT NULL,
    root_cause_analysis TEXT NOT NULL,
    five_whys JSONB DEFAULT '[]', -- Array of {"why": string, "answer": string}
    chronological_timeline JSONB DEFAULT '[]',
    preventive_actions JSONB DEFAULT '[]',
    published BOOLEAN NOT NULL DEFAULT FALSE,
    author_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 8. RCA Action Items Table
CREATE TABLE rca_action_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rca_id UUID NOT NULL REFERENCES rca_reports(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date DATE,
    status VARCHAR(50) NOT NULL DEFAULT 'open', -- 'open', 'in_progress', 'completed'
    external_ticket_url VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

# Part 2: Monolithic Route Specifications

## 1. Web UI Routes (HTML Template Rendering)

Rendered via Go `html/template` files located in `templates/`:

| Method | Route | Template Rendered | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/` | `templates/pages/dashboard.html` | SRE Dashboard: Active incidents, alert counters, severity breakdown |
| `GET` | `/incidents` | `templates/pages/incidents.html` | Searchable & filterable incident triage list |
| `GET` | `/incidents/:id` | `templates/pages/incident_detail.html` | Investigation Workspace: Real-time timeline, correlated alerts & AI Copilot |
| `GET` | `/knowledge` | `templates/pages/knowledge.html` | Knowledge Explorer: Runbooks, search query input, vector search results |
| `GET` | `/incidents/:id/rca` | `templates/pages/rca_editor.html` | Postmortem editor, 5-Whys visualizer, action items |
| `GET` | `/rca/:id` | `templates/pages/rca_view.html` | Published RCA view and printable report |

---

## 2. Real-Time Streaming & HTMX Partial Endpoints

| Method | Route | Output Format | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/incidents/:id/sse` | `text/event-stream` | SSE stream broadcasting new alerts and status changes to the client |
| `POST` | `/incidents/:id/ai/chat` | `text/event-stream` | Streams AI Copilot responses token-by-token directly to the investigation UI |
| `POST` | `/incidents/:id/timeline/add` | HTML Partial (`timeline_item.html`) | Adds a manual note to the incident timeline via HTMX |
| `PATCH` | `/incidents/:id/status` | HTML Partial (`status_badge.html`) | Updates incident status and returns updated badge via HTMX |

---

## 3. Ingestion Webhook APIs

### `POST /api/v1/webhooks/alerts/prometheus`
Ingests standard Prometheus / Alertmanager v4 webhook payloads.
- **Request Payload:**
```json
{
  "version": "4",
  "groupKey": "{}:{alertname=\"DatabasePoolExhaustion\"}",
  "status": "firing",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "DatabasePoolExhaustion",
        "service": "order-service",
        "environment": "production",
        "severity": "P1"
      },
      "annotations": {
        "summary": "Active connections reached 98% of pool capacity",
        "description": "Postgres connection pool max_connections reached for order-service replicas."
      },
      "startsAt": "2026-11-25T11:15:00Z"
    }
  ]
}
```
- **Responses:**
  - `202 Accepted`: `{ "status": "queued", "processed_alerts": 1 }`
  - `400 Bad Request`: Validation failure.

---

## 4. AI & Knowledge REST Endpoints

### `POST /api/v1/incidents/:id/ai/triage`
Triggers AI incident synthesis, root cause analysis, and similar incident lookup.
- **Response `200 OK`:**
```json
{
  "incident_id": "7b8f9e12-3456-4c78-90ab-cdef12345678",
  "summary": "Order-service is experiencing database connection exhaustion triggering upstream HTTP 504 gateway timeouts.",
  "probable_root_cause": "Database connection pool leak following deployment or unexpected spike in lock contention on table `orders`.",
  "recommended_actions": [
    {
      "step": 1,
      "action": "Inspect active DB queries and lock waits",
      "command": "SELECT pid, query, state, age(clock_timestamp(), query_start) FROM pg_stat_activity WHERE state != 'idle' ORDER BY age DESC LIMIT 10;"
    },
    {
      "step": 2,
      "action": "Restart hung application replicas or bump max_pool_size temporarily",
      "command": "kubectl rollout restart deployment/order-service -n production"
    }
  ],
  "similar_incidents": [
    {
      "incident_id": "2a1e3c54-7890-4d12-89ef-abcdef987654",
      "incident_number": 88,
      "title": "Postgres connection spike due to missing index on orders",
      "similarity_score": 0.89,
      "resolution": "Rolled back commit and added composite index on (user_id, status)."
    }
  ]
}
```

### `POST /api/v1/knowledge/search`
Semantic search across runbooks and past incidents using `pgvector`.
- **Request Body:**
```json
{
  "query": "How to resolve database pool exhaustion on order-service?",
  "service_id": "Optional UUID",
  "top_k": 3
}
```
