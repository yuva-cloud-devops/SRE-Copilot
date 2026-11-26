# Product Requirements Document (PRD)
## SRE Copilot: Intelligent Application Incident Response Platform

- **Document Version:** 1.1.0
- **Status:** Approved / Architecture Ready
- **Architecture:** Monolithic Go Application (`html/template` UI + Embedded Backend)
- **Primary Stakeholders:** SRE Teams, DevOps/Platform Engineering, Production Support

---

## 1. Problem Statement & Motivation

Modern engineering organizations operate complex distributed microservice environments. When an outage occurs, on-call engineers are inundated with alerts across multiple tools (Prometheus, Grafana, PagerDuty, GitHub, Confluence, Jira). 

### Key Pain Points:
1. **Alert Fatigue & Noise:** Cascading failures create alert storms (e.g., downstream 504 gateway timeouts mask upstream database pool exhaustion).
2. **Context Fragmentation:** Engineers waste 15–30 minutes gathering context across metrics dashboards, logs, recent code deployments, and documentation.
3. **Repetitive Triage:** Known issues are re-investigated from scratch due to siloed or unsearchable postmortems.
4. **Outdated Runbooks:** Existing documentation is static, rarely updated, and detached from the operational workflow.
5. **Slow RCA Preparation:** Drafting post-incident root cause analyses (RCA) and timelines takes hours of manual transcript assembly.

---

## 2. Product Vision & Architectural Philosophy

**SRE Copilot** is designed as a **self-contained Go monolith**. SREs value simplicity, fast startup times, low operational overhead, and minimal runtime dependencies. 

By delivering the frontend using **Go `html/template`** (augmented with HTMX and Tailwind CSS) packaged inside the same Go binary:
- **No Node.js runtime required** in production or CI/CD pipelines.
- **Single executable binary** that can run directly on a VM, in a minimal distroless Docker container, or Kubernetes pod.
- **Ultra-low latency rendering** directly from memory with unified request lifecycles.

The platform answers the four core operational questions during an incident:
1. **What is broken?** (Automated incident synthesis and impact radius)
2. **Why is it broken?** (Root cause hypothesis and correlation of alerts/deployments)
3. **What should I do next?** (Contextual investigation steps and verified runbook recommendations)
4. **How can I prevent it from happening again?** (Automated RCA drafting, preventive action tracking)

---

## 3. Target Personas

| Persona | Role | Key Pain Point | Primary MVP Feature Used |
| :--- | :--- | :--- | :--- |
| **Alex - On-Call SRE** | Primary Responder | Context switching during high-severity outages | Incident Timeline, AI Copilot Next Steps, Alert Grouping |
| **Priya - Platform Engineer** | Infrastructure & Service Owner | Identifying which upstream dependency caused failure | Service-centric correlation, Prometheus/GitHub integration |
| **Marcus - Incident Commander** | Lead Coordinator | Aggregating status updates and timelines for stakeholders | Auto-generated incident summaries, RCA Center |
| **Elena - Engineering Manager** | Team Lead | Action item tracking and recurring failure prevention | RCA action item tracking, historical incident trends |

---

## 4. MVP Scope Definition

### In-Scope (Phase 1 MVP)

1. **Incident Lifecycle Management:**
   - Automatic incident creation via alert triggers or manual creation.
   - Status transitions (`Investigating`, `Identified`, `Monitoring`, `Resolved`).
   - Standardized severities (`P1 - Critical`, `P2 - High`, `P3 - Medium`, `P4 - Low`).
   - Unified real-time incident event timeline via Server-Sent Events (SSE).

2. **Alert & Log Ingestion via Google Cloud Platform (GCP):**
   - Webhook receivers for **GCP Cloud Monitoring Alerting Policies** (`/api/v1/webhooks/gcp/monitoring`).
   - Webhook receivers for **GCP Cloud Logging Sinks** (`/api/v1/webhooks/gcp/logging`) for real-time error log ingestion.
   - Rule-based & heuristic alert grouping (Sliding time window, service/resource name, environment).
   - In-process sliding-window deduplication and noise suppression (no Redis dependency).

3. **AI Investigation Copilot:**
   - Real-time incident summarization.
   - Root cause hypothesis generation based on incoming alerts and service metadata.
   - Dynamic next-step investigation prompts (e.g., specific metrics queries or log filters).
   - Retrieval of top-3 similar historical incidents with resolution notes.
   - Streaming token response rendered in the Go HTML template interface.

4. **Knowledge Base & Semantic Search:**
   - Markdown runbook indexing with vector embeddings (`pgvector`).
   - Natural language search across historical incidents, runbooks, and RCAs.

5. **Automated RCA Generation:**
   - 1-click RCA report generation including: Summary, Impact, Chronological Timeline, 5 Whys, Root Cause, Action Items.
   - Export to Markdown and formatted HTML print/PDF.

### Out-of-Scope (Future Phases)
- Autonomous remediation / auto-healing scripts (Phase 4).
- Deep Kubernetes pod manipulation / cluster kernel profiling (Phase 3).
- Automated CI/CD deployment rollbacks (Phase 3).
- Live audio/call transcription for war rooms (Phase 2).
- Native bi-directional Slack / Teams bot (Phase 2).

---

## 5. Functional Requirements & User Stories

### Epic 1: Incident Ingestion & Management
- **US1.1:** As an SRE, I want alerts from Prometheus/Alertmanager to automatically create or be linked to an active incident so that I don't have to manually create tickets.
  - *Acceptance Criteria:* Ingestion webhook accepts Alertmanager v4 JSON format; response time < 100ms.
- **US1.2:** As an Incident Commander, I want to update incident severity and status with an audit trail so stakeholders have live visibility.
  - *Acceptance Criteria:* Changes emit a timeline event and push updates to the UI in real time via Server-Sent Events (SSE).

### Epic 2: Alert Correlation & Noise Reduction
- **US2.1:** As an on-call engineer, I want duplicate alerts within a 15-minute rolling window to be grouped into an existing incident rather than paging me multiple times.
  - *Acceptance Criteria:* Alerts matching `(service_id, environment, alert_name)` within window `T` increment recurrence count without spawning duplicate incidents.

### Epic 3: AI-Assisted Investigation
- **US3.1:** As an engineer, I want an AI-generated natural language summary explaining what is failing across the correlated alerts.
  - *Acceptance Criteria:* Summary generated within 3 seconds of incident creation; cites specific alerts and affected services.
- **US3.2:** As an engineer, I want the Copilot to recommend 3 concrete diagnostic steps and relevant runbook sections.
  - *Acceptance Criteria:* Recommendations provide exact command/query snippets and link to verified runbook entries.

### Epic 4: Knowledge Base & Vector Search
- **US4.1:** As an engineer, I want to search for similar past incidents using conversational questions (e.g., "checkout service connection pool exhaustion").
  - *Acceptance Criteria:* Vector similarity search returns top-5 relevant past incidents with cosine similarity score > 0.75.

### Epic 5: Post-Incident RCA Generation
- **US5.1:** As an Incident Commander, I want to generate a complete draft RCA with a pre-populated timeline with a single click once an incident is marked `Resolved`.
  - *Acceptance Criteria:* Draft contains: executive summary, timeline of state transitions and alerts, suspected root cause, and initial corrective action items.

---

## 6. Non-Functional Requirements (NFRs)

| Category | Requirement | Target Metric |
| :--- | :--- | :--- |
| **Footprint** | Single binary distribution | Single Go binary, < 50MB, zero node runtime |
| **Performance** | Server-side template render latency | < 30ms TTFB for all pages |
| **Performance** | AI Copilot streaming response (TTFT) | < 1.2s to first token |
| **Availability** | System uptime during business operations | 99.9% availability |
| **Scalability** | Alert ingestion throughput | 1,000 alerts/sec with zero message loss |
| **Security** | Authentication & Access Control | Session/JWT auth, Role-Based Access Control |
| **Data Retention**| Alert & timeline retention | 90 days active hot storage; RCAs retained indefinitely |

---

## 7. Success Metrics & KPIs

1. **MTTR (Mean Time to Resolution):** 35% decrease measured over 60 days.
2. **Alert-to-Incident Ratio:** Target reduction from 20:1 to 3:1 via correlation.
3. **Copilot Relevance Score:** ≥ 80% positive feedback (thumbs up) on suggested investigation steps.
4. **RCA Time-to-Publish:** Reduced from an average of 4 days post-incident to < 24 hours.
