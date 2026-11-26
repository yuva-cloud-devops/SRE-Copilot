package repository

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"sre-copilot/internal/models"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	incidents   map[string]*models.Incident
	alerts      map[string]*models.Alert
	logs        map[string][]models.GCPLogRecord
	timeline    map[string][]models.TimelineEvent
	knowledge   map[string]*models.KnowledgeDocument
	rcas        map[string]*models.RCAReport
	incidentSeq int
}

func NewMemoryRepository() *MemoryRepository {
	repo := &MemoryRepository{
		incidents:   make(map[string]*models.Incident),
		alerts:      make(map[string]*models.Alert),
		logs:        make(map[string][]models.GCPLogRecord),
		timeline:    make(map[string][]models.TimelineEvent),
		knowledge:   make(map[string]*models.KnowledgeDocument),
		rcas:        make(map[string]*models.RCAReport),
		incidentSeq: 100,
	}
	repo.seedInitialData()
	return repo
}

func (r *MemoryRepository) seedInitialData() {
	now := time.Now()

	// Seed GCP Operational Runbooks
	runbook1 := &models.KnowledgeDocument{
		ID:           "kb-gcp-001",
		Title:        "Runbook: GCP Cloud Run Container High Memory & OOM",
		DocumentType: "runbook",
		ServiceName:  "order-service",
		Content: `# Runbook: GCP Cloud Run Container High Memory & OOM

## Symptoms
- GCP Alert: "Container memory utilization above 90%"
- Metric: run.googleapis.com/container/memory/utilizations
- Cloud Logging error: "Memory limit of 512MiB exceeded with 514MiB used. Consider increasing the memory limit"

## Investigation in GCP Cloud Console
1. Check Cloud Logging filter:
   resource.type="cloud_run_revision"
   resource.labels.service_name="order-service"
   severity>=ERROR

2. Inspect current revision resource limits:
   gcloud run services describe order-service --region=us-central1 --format='value(spec.template.spec.containers[0].resources.limits.memory)'

## Immediate Mitigation
1. Increase Cloud Run container memory limit:
   gcloud run services update order-service --memory=1Gi --region=us-central1
2. Deploy revision with leak patch or restart hung container instances.`,
		Tags:      []string{"gcp", "cloud_run", "memory", "oom", "p1", "order-service"},
		CreatedAt: now.Add(-30 * 24 * time.Hour),
	}

	runbook2 := &models.KnowledgeDocument{
		ID:           "kb-gcp-002",
		Title:        "Runbook: GCP Cloud SQL Postgres High Connection Saturation",
		DocumentType: "runbook",
		ServiceName:  "order-service",
		Content: `# Runbook: GCP Cloud SQL Postgres Connection Saturation

## Symptoms
- GCP Alert: "Cloud SQL Postgres connection pool > 95%"
- Metric: cloudsql.googleapis.com/database/postgresql/num_backends
- HTTP 504 timeouts on upstream Cloud Run / GKE services

## Mitigation
1. Check active database connections in Cloud Logging:
   SELECT pid, query, state, age(clock_timestamp(), query_start) 
   FROM pg_stat_activity WHERE state != 'idle' ORDER BY age DESC LIMIT 10;
2. Terminate blocking backend PID via Cloud SQL client.`,
		Tags:      []string{"gcp", "cloud_sql", "postgres", "pool", "p1"},
		CreatedAt: now.Add(-20 * 24 * time.Hour),
	}

	r.knowledge[runbook1.ID] = runbook1
	r.knowledge[runbook2.ID] = runbook2

	// Seed Sample GCP Incident
	incID := "inc-101"
	r.incidentSeq++
	incident := &models.Incident{
		ID:                  incID,
		IncidentNumber:      101,
		Title:               "GCP Cloud Run Memory Saturation & OOM on order-service",
		Summary:             "GCP Alerting Policy fired: container memory utilization exceeded 92% on revision order-service-00042.",
		ServiceName:         "order-service",
		Environment:         "production",
		Severity:            models.SeverityP1,
		Status:              models.StatusInvestigating,
		LeadResponder:       "Alex (On-Call SRE)",
		RootCauseHypothesis: "Heap growth in order-service container exceeding 512MiB limit due to unbuffered JSON payload parsing.",
		StartedAt:           now.Add(-25 * time.Minute),
		CreatedAt:           now.Add(-25 * time.Minute),
		UpdatedAt:           now.Add(-5 * time.Minute),
		AlertCount:          2,
	}

	alert1 := models.Alert{
		ID:            "alt-gcp-001",
		IncidentID:    incID,
		Fingerprint:   "order-service-gcp-mem-prod",
		AlertName:     "Cloud Run Container Memory Utilization > 90%",
		Source:        "gcp_monitoring",
		ServiceName:   "order-service",
		Environment:   "production",
		Severity:      models.SeverityP1,
		Status:        models.AlertStatusFiring,
		Description:   "Memory utilization for cloud_run_revision order-service reached 94.2% (threshold: 90.0%).",
		GCPPolicyName: "Cloud Run Production Memory Alerts",
		GCPCondition:  "Container memory utilization above 90%",
		GCPMetricType: "run.googleapis.com/container/memory/utilizations",
		GCPConsoleURL: "https://console.cloud.google.com/monitoring/alerting",
		Labels: map[string]string{
			"project_id":   "prod-ecommerce-project",
			"service_name": "order-service",
			"location":     "us-central1",
		},
		RecurrenceCount: 6,
		TriggeredAt:     now.Add(-25 * time.Minute),
		LastSeenAt:      now.Add(-2 * time.Minute),
	}

	log1 := models.GCPLogRecord{
		ID:          "log-gcp-001",
		IncidentID:  incID,
		LogName:     "projects/prod-ecommerce-project/logs/run.googleapis.com%2Fstderr",
		Severity:    "ERROR",
		ServiceName: "order-service",
		Message:     "Memory limit of 512MiB exceeded with 514MiB used. Container terminated by cgroup OOM killer.",
		Timestamp:   now.Add(-24 * time.Minute),
	}

	r.incidents[incID] = incident
	r.alerts[alert1.ID] = &alert1
	r.logs[incID] = append(r.logs[incID], log1)

	r.timeline[incID] = []models.TimelineEvent{
		{
			ID:          "tl-001",
			IncidentID:  incID,
			EventType:   models.EventAlertTriggered,
			Description: "GCP Alerting Policy 'Cloud Run Production Memory Alerts' fired for order-service [us-central1].",
			CreatedAt:   now.Add(-25 * time.Minute),
		},
		{
			ID:          "tl-002",
			IncidentID:  incID,
			EventType:   models.EventLogCorrelated,
			Description: "Cloud Logging ERROR: 'Memory limit of 512MiB exceeded with 514MiB used. Container terminated by cgroup OOM killer.'",
			CreatedAt:   now.Add(-24 * time.Minute),
		},
		{
			ID:          "tl-003",
			IncidentID:  incID,
			EventType:   models.EventSeverityChanged,
			Description: "Incident auto-triaged to P1 Critical.",
			CreatedAt:   now.Add(-25 * time.Minute),
		},
		{
			ID:          "tl-004",
			IncidentID:  incID,
			EventType:   models.EventCommentAdded,
			Description: "Checking recent revision diffs. Scaling memory limit via gcloud run to 1Gi as temporary buffer.",
			CreatedBy:   "Alex (On-Call SRE)",
			CreatedAt:   now.Add(-10 * time.Minute),
		},
	}
}

func (r *MemoryRepository) ListIncidents(status models.IncidentStatus, severity models.IncidentSeverity) ([]models.Incident, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []models.Incident
	for _, inc := range r.incidents {
		if status != "" && inc.Status != status {
			continue
		}
		if severity != "" && inc.Severity != severity {
			continue
		}
		result = append(result, *inc)
	}
	return result, nil
}

func (r *MemoryRepository) GetIncident(id string) (*models.Incident, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inc, exists := r.incidents[id]
	if !exists {
		return nil, fmt.Errorf("incident not found: %s", id)
	}

	copied := *inc
	for _, a := range r.alerts {
		if a.IncidentID == id {
			copied.Alerts = append(copied.Alerts, *a)
		}
	}
	copied.Logs = r.logs[id]
	copied.Timeline = r.timeline[id]
	copied.AlertCount = len(copied.Alerts)

	return &copied, nil
}

func (r *MemoryRepository) CreateIncident(inc *models.Incident) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.incidentSeq++
	inc.IncidentNumber = r.incidentSeq
	inc.CreatedAt = time.Now()
	inc.UpdatedAt = time.Now()
	r.incidents[inc.ID] = inc
	return nil
}

func (r *MemoryRepository) UpdateIncident(inc *models.Incident) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.incidents[inc.ID]; !exists {
		return fmt.Errorf("incident not found: %s", inc.ID)
	}
	inc.UpdatedAt = time.Now()
	r.incidents[inc.ID] = inc
	return nil
}

func (r *MemoryRepository) AddTimelineEvent(event *models.TimelineEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	r.timeline[event.IncidentID] = append(r.timeline[event.IncidentID], *event)
	return nil
}

func (r *MemoryRepository) GetTimeline(incidentID string) ([]models.TimelineEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := r.timeline[incidentID]
	return events, nil
}

func (r *MemoryRepository) AddAlert(alert *models.Alert) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.alerts[alert.ID] = alert
	return nil
}

func (r *MemoryRepository) GetAlerts(incidentID string) ([]models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts []models.Alert
	for _, a := range r.alerts {
		if a.IncidentID == incidentID {
			alerts = append(alerts, *a)
		}
	}
	return alerts, nil
}

func (r *MemoryRepository) AddLog(record *models.GCPLogRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logs[record.IncidentID] = append(r.logs[record.IncidentID], *record)
	return nil
}

func (r *MemoryRepository) GetLogs(incidentID string) ([]models.GCPLogRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.logs[incidentID], nil
}

func (r *MemoryRepository) FindOpenIncidentForAlert(serviceName, env string, windowMinutes int) (*models.Incident, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cutoff := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)

	for _, inc := range r.incidents {
		if inc.Status != models.StatusResolved &&
			strings.EqualFold(inc.ServiceName, serviceName) &&
			strings.EqualFold(inc.Environment, env) &&
			inc.UpdatedAt.After(cutoff) {
			return inc, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepository) Search(query string, serviceName string, limit int) ([]models.KnowledgeDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := strings.ToLower(query)
	var matches []models.KnowledgeDocument

	for _, doc := range r.knowledge {
		if serviceName != "" && !strings.EqualFold(doc.ServiceName, serviceName) {
			continue
		}
		titleMatch := strings.Contains(strings.ToLower(doc.Title), q)
		contentMatch := strings.Contains(strings.ToLower(doc.Content), q)
		tagMatch := false
		for _, tag := range doc.Tags {
			if strings.Contains(strings.ToLower(tag), q) {
				tagMatch = true
				break
			}
		}

		if titleMatch || contentMatch || tagMatch || q == "" {
			matches = append(matches, *doc)
			if len(matches) >= limit {
				break
			}
		}
	}
	return matches, nil
}

func (r *MemoryRepository) ListAll() ([]models.KnowledgeDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []models.KnowledgeDocument
	for _, doc := range r.knowledge {
		all = append(all, *doc)
	}
	return all, nil
}

func (r *MemoryRepository) GetDocument(id string) (*models.KnowledgeDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doc, exists := r.knowledge[id]
	if !exists {
		return nil, fmt.Errorf("document not found: %s", id)
	}
	return doc, nil
}

func (r *MemoryRepository) AddDocument(doc *models.KnowledgeDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.knowledge[doc.ID] = doc
	return nil
}

func (r *MemoryRepository) GetRCA(incidentID string) (*models.RCAReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rca, exists := rcasMap[incidentID]
	if !exists {
		return nil, nil
	}
	return rca, nil
}

var rcasMap = make(map[string]*models.RCAReport)

func (r *MemoryRepository) SaveRCA(rca *models.RCAReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rcasMap[rca.IncidentID] = rca
	return nil
}

func (r *MemoryRepository) ListRCAs() ([]models.RCAReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []models.RCAReport
	for _, rca := range rcasMap {
		result = append(result, *rca)
	}
	return result, nil
}
