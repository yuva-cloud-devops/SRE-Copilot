# SRE Copilot

> AI-Powered Incident Intelligence & Operational Copilot for SRE and Production Engineering Teams.

SRE Copilot unifies alerts, timelines, runbooks, and historical postmortems into an automated assistant answering four essential operational questions:
1. **What is broken?** (Incident synthesis & impact radius)
2. **Why is it broken?** (Root cause hypothesis & alert correlation)
3. **What should I do next?** (Diagnostic actions & runbook guidance)
4. **How can I prevent it from happening again?** (Automated RCA drafting & action item tracking)

---

## 🏗 Architecture Style: Pure Go Monolith

The application is structured as a **single, self-contained Go monolithic binary**:
- **Backend & Core Logic:** Go with Gin Web Framework (or standard library router)
- **Frontend Presentation:** Server-side rendered HTML using Go `html/template` located in the `templates/` directory, styled with modern Tailwind CSS and enhanced with HTMX / Alpine.js for real-time reactivity (SSE live timeline, streaming Copilot chat, and dynamic modals).
- **Zero Node.js dependency:** No separate Node.js build pipeline or frontend server required. Easy single-binary packaging and deployment.
- **Database:** PostgreSQL with native SQL driver (`github.com/lib/pq`) and graceful in-memory fallback
- **Ingestion Sources:** Google Cloud Platform (GCP Cloud Monitoring Alerting Policies + GCP Cloud Logging Export Sinks)
- **Cache & Real-Time:** In-process Go channels for real-time Server-Sent Events (SSE) and sliding-window deduplication (Zero Redis dependency)
- **Configuration:** Managed via root `config.json` with environment variable overrides
- **AI Engine:** Multi-provider support (Google Gemini, OpenAI, or integrated offline Copilot simulator)

---

## 📚 Core Architecture & Design Documents

1. **[Product Requirements Document (PRD)](file:///d:/SRE-Copilot/docs/PRD.md)**
   - Problem statement, user personas, MVP scope definition, functional user stories & acceptance criteria, non-functional requirements, and operational KPIs (MTTR, alert noise reduction).

2. **[High-Level System Design (HLD)](file:///d:/SRE-Copilot/docs/HLD.md)**
   - Monolithic architecture diagram (Mermaid), directory layout, sliding-window alert correlation engine, AI RAG pipeline, and SSE real-time streaming.

3. **[Database Schema & API / Route Specification](file:///d:/SRE-Copilot/docs/DATABASE_AND_API_SPEC.md)**
   - PostgreSQL 16 + `pgvector` DDL with HNSW vector indexes, UI template route mappings, and complete RESTful/SSE webhook & Copilot endpoints.

---

## 📁 Monolithic Project Structure Preview

```
d:/SRE-Copilot/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── docs/                        # Architecture & PRD documentation
├── internal/
│   ├── config/                  # Environment & app configuration
│   ├── database/                # Postgres (pgx/pgvector) & Redis connections
│   ├── models/                  # Domain structs (Incident, Alert, RCA, etc.)
│   ├── repository/              # SQL queries & vector search operations
│   ├── services/
│   │   ├── alert_correlator.go  # Sliding-window correlation & deduplication
│   │   ├── incident_service.go  # Incident lifecycle & timeline events
│   │   ├── ai_copilot.go        # Prompt orchestration, RAG, LLM client
│   │   └── knowledge_service.go # Runbook embedding & pgvector retrieval
│   └── handlers/
│       ├── web_handlers.go      # HTML template rendering handlers
│       ├── api_handlers.go      # REST & SSE endpoints
│       └── webhook_handlers.go  # Prometheus / Alertmanager ingestion
├── templates/                   # Go html/template files
│   ├── layouts/
│   │   └── base.html            # App shell, navigation, header, scripts
│   ├── partials/
│   │   ├── timeline_item.html   # Live SSE timeline updates
│   │   ├── alert_card.html      # Grouped alert components
│   │   └── copilot_chat.html    # Streaming response partials
│   └── pages/
│       ├── dashboard.html       # Incident triage & metrics overview
│       ├── incident_detail.html # Split-screen investigation workspace
│       ├── knowledge.html       # Runbook & postmortem search
│       └── rca_detail.html      # Postmortem editor & action item manager
├── static/                      # Static assets (Tailwind CSS, JS, SVGs)
│   ├── css/
│   ├── js/
│   └── img/
├── go.mod
└── README.md
```
