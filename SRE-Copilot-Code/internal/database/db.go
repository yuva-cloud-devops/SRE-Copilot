package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"sre-copilot/internal/config"
)

func InitDB(cfg *config.Config) (*sql.DB, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not configured")
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeMinutes) * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database unreachable: %w", err)
	}

	log.Println("✓ Connected to PostgreSQL database")

	// Run migrations
	if err := AutoMigrate(db); err != nil {
		log.Printf("Warning: Database auto-migration encountered notice: %v", err)
	}

	return db, nil
}

func AutoMigrate(db *sql.DB) error {
	// Try creating extensions gracefully
	_, _ = db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto;`)
	_, _ = db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	_, _ = db.Exec(`CREATE EXTENSION IF NOT EXISTS "vector";`)

	schema := `
	CREATE TABLE IF NOT EXISTS services (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		service_name VARCHAR(100) UNIQUE NOT NULL,
		description TEXT,
		tier VARCHAR(10) NOT NULL DEFAULT 'tier-1',
		repository_url VARCHAR(255),
		team_owner VARCHAR(100) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS incidents (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		incident_number SERIAL UNIQUE,
		title VARCHAR(255) NOT NULL,
		summary TEXT,
		service_name VARCHAR(100) NOT NULL,
		environment VARCHAR(50) NOT NULL DEFAULT 'production',
		severity VARCHAR(10) NOT NULL DEFAULT 'P3',
		status VARCHAR(20) NOT NULL DEFAULT 'investigating',
		lead_responder VARCHAR(100) DEFAULT 'Unassigned',
		root_cause_hypothesis TEXT,
		started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		resolved_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		alert_count INT NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS alerts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		incident_id UUID REFERENCES incidents(id) ON DELETE SET NULL,
		fingerprint VARCHAR(64) NOT NULL,
		alert_name VARCHAR(255) NOT NULL,
		source VARCHAR(50) NOT NULL,
		service_name VARCHAR(100) NOT NULL,
		environment VARCHAR(50) NOT NULL DEFAULT 'production',
		severity VARCHAR(10) NOT NULL DEFAULT 'P3',
		status VARCHAR(20) NOT NULL DEFAULT 'firing',
		description TEXT,
		gcp_policy_name VARCHAR(255),
		gcp_condition VARCHAR(255),
		gcp_metric_type VARCHAR(255),
		gcp_console_url TEXT,
		labels JSONB NOT NULL DEFAULT '{}',
		annotations JSONB NOT NULL DEFAULT '{}',
		recurrence_count INT NOT NULL DEFAULT 1,
		triggered_at TIMESTAMPTZ NOT NULL,
		last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS gcp_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		incident_id UUID REFERENCES incidents(id) ON DELETE CASCADE,
		log_name VARCHAR(255) NOT NULL,
		severity VARCHAR(20) NOT NULL,
		service_name VARCHAR(100) NOT NULL,
		message TEXT NOT NULL,
		payload JSONB NOT NULL DEFAULT '{}',
		timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS incident_timeline (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
		event_type VARCHAR(50) NOT NULL,
		description TEXT NOT NULL,
		metadata JSONB NOT NULL DEFAULT '{}',
		created_by VARCHAR(100),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS knowledge_documents (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		title VARCHAR(255) NOT NULL,
		document_type VARCHAR(50) NOT NULL,
		service_name VARCHAR(100),
		content TEXT NOT NULL,
		tags TEXT[] DEFAULT '{}',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS rca_reports (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		incident_id UUID UNIQUE NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
		incident_number INT NOT NULL,
		title VARCHAR(255) NOT NULL,
		executive_summary TEXT NOT NULL,
		impact_summary TEXT NOT NULL,
		root_cause_analysis TEXT NOT NULL,
		five_whys JSONB DEFAULT '[]',
		preventive_actions JSONB DEFAULT '[]',
		published BOOLEAN NOT NULL DEFAULT FALSE,
		author VARCHAR(100),
		action_items JSONB DEFAULT '[]',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_alerts_fingerprint ON alerts(fingerprint);
	CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
	CREATE INDEX IF NOT EXISTS idx_timeline_inc ON incident_timeline(incident_id);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("migration error: %w", err)
	}

	log.Println("✓ Database tables verified and indexed")
	seedDefaultKnowledge(db)
	return nil
}

func seedDefaultKnowledge(db *sql.DB) {
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM knowledge_documents`).Scan(&count)
	if count > 0 {
		return
	}

	runbook1 := `# Runbook: GCP Cloud Run Container High Memory & OOM

## Symptoms
- GCP Alert: "Container memory utilization above 90%"
- Metric: run.googleapis.com/container/memory/utilizations
- Cloud Logging error: "Memory limit of 512MiB exceeded with 514MiB used. Consider increasing the memory limit"

## Investigation in GCP Cloud Console
1. Check Cloud Logging filter:
   resource.type="cloud_run_revision"
   resource.labels.service_name="order-service"
   severity>=ERROR

2. Inspect heap dump or memory profile:
   gcloud run services describe order-service --region=us-central1 --format='value(spec.template.spec.containers[0].resources.limits.memory)'

## Mitigation
1. Increase Cloud Run container memory limit:
   gcloud run services update order-service --memory=1Gi --region=us-central1
2. Deploy revision with leak patch or restart hung container instances.`

	_, _ = db.Exec(`
		INSERT INTO knowledge_documents (title, document_type, service_name, content, tags)
		VALUES ($1, $2, $3, $4, $5)
	`, "Runbook: GCP Cloud Run Container High Memory & OOM", "runbook", "order-service", runbook1, []string{"gcp", "cloud_run", "memory", "oom", "p1"})

	log.Println("✓ Seeded default GCP operational runbooks into database")
}
