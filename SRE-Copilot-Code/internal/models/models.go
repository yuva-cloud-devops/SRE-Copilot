package models

import (
	"time"
)

type IncidentSeverity string

const (
	SeverityP1 IncidentSeverity = "P1"
	SeverityP2 IncidentSeverity = "P2"
	SeverityP3 IncidentSeverity = "P3"
	SeverityP4 IncidentSeverity = "P4"
)

type IncidentStatus string

const (
	StatusInvestigating IncidentStatus = "investigating"
	StatusIdentified    IncidentStatus = "identified"
	StatusMonitoring    IncidentStatus = "monitoring"
	StatusResolved      IncidentStatus = "resolved"
)

type AlertStatus string

const (
	AlertStatusFiring     AlertStatus = "firing"
	AlertStatusResolved   AlertStatus = "resolved"
	AlertStatusSuppressed AlertStatus = "suppressed"
)

type EventType string

const (
	EventAlertTriggered  EventType = "alert_triggered"
	EventStatusChanged   EventType = "status_changed"
	EventSeverityChanged EventType = "severity_changed"
	EventCommentAdded    EventType = "comment_added"
	EventRunbookExecuted EventType = "runbook_executed"
	EventAIInsight       EventType = "ai_insight"
	EventLogCorrelated   EventType = "log_correlated"
)

// Service represents an application/service in the microservice ecosystem
type Service struct {
	ID            string    `json:"id"`
	ServiceName   string    `json:"service_name"`
	Description   string    `json:"description"`
	Tier          string    `json:"tier"` // tier-1, tier-2, tier-3
	RepositoryURL string    `json:"repository_url"`
	TeamOwner     string    `json:"team_owner"`
	CreatedAt     time.Time `json:"created_at"`
}

// Alert represents an ingested monitoring alert from GCP Cloud Monitoring
type Alert struct {
	ID              string            `json:"id"`
	IncidentID      string            `json:"incident_id,omitempty"`
	Fingerprint     string            `json:"fingerprint"`
	AlertName       string            `json:"alert_name"`
	Source          string            `json:"source"` // "gcp_monitoring", "gcp_logging"
	ServiceName     string            `json:"service_name"`
	Environment     string            `json:"environment"`
	Severity        IncidentSeverity  `json:"severity"`
	Status          AlertStatus       `json:"status"`
	Description     string            `json:"description"`
	GCPPolicyName   string            `json:"gcp_policy_name,omitempty"`
	GCPCondition    string            `json:"gcp_condition,omitempty"`
	GCPMetricType   string            `json:"gcp_metric_type,omitempty"`
	GCPConsoleURL   string            `json:"gcp_console_url,omitempty"`
	Labels          map[string]string `json:"labels"`
	Annotations     map[string]string `json:"annotations"`
	RecurrenceCount int               `json:"recurrence_count"`
	TriggeredAt     time.Time         `json:"triggered_at"`
	LastSeenAt      time.Time         `json:"last_seen_at"`
}

// GCPLogRecord represents a critical error or anomaly log correlated to an incident
type GCPLogRecord struct {
	ID          string                 `json:"id"`
	IncidentID  string                 `json:"incident_id"`
	LogName     string                 `json:"log_name"`
	Severity    string                 `json:"severity"` // ERROR, CRITICAL, WARNING
	ServiceName string                 `json:"service_name"`
	Message     string                 `json:"message"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// TimelineEvent records any state change or activity in an incident
type TimelineEvent struct {
	ID          string                 `json:"id"`
	IncidentID  string                 `json:"incident_id"`
	EventType   EventType              `json:"event_type"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// Incident represents an active or past production incident
type Incident struct {
	ID                  string            `json:"id"`
	IncidentNumber      int               `json:"incident_number"`
	Title               string            `json:"title"`
	Summary             string            `json:"summary"`
	ServiceName         string            `json:"service_name"`
	Environment         string            `json:"environment"`
	Severity            IncidentSeverity  `json:"severity"`
	Status              IncidentStatus    `json:"status"`
	LeadResponder       string            `json:"lead_responder"`
	RootCauseHypothesis string            `json:"root_cause_hypothesis"`
	StartedAt           time.Time         `json:"started_at"`
	ResolvedAt          *time.Time        `json:"resolved_at,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	AlertCount          int               `json:"alert_count"`
	Alerts              []Alert           `json:"alerts,omitempty"`
	Logs                []GCPLogRecord    `json:"logs,omitempty"`
	Timeline            []TimelineEvent   `json:"timeline,omitempty"`
}

// KnowledgeDocument represents a runbook, architecture guide, or past postmortem
type KnowledgeDocument struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	DocumentType string    `json:"document_type"` // 'runbook', 'historical_rca', 'architecture'
	ServiceName  string    `json:"service_name"`
	Content      string    `json:"content"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
}

// RCAReport represents the postmortem root cause analysis
type RCAReport struct {
	ID                    string         `json:"id"`
	IncidentID            string         `json:"incident_id"`
	IncidentNumber        int            `json:"incident_number"`
	Title                 string         `json:"title"`
	ExecutiveSummary      string         `json:"executive_summary"`
	ImpactSummary         string         `json:"impact_summary"`
	RootCauseAnalysis     string         `json:"root_cause_analysis"`
	FiveWhys              []FiveWhyItem  `json:"five_whys"`
	PreventiveActions     []string       `json:"preventive_actions"`
	Published             bool           `json:"published"`
	Author                string         `json:"author"`
	ActionItems           []ActionItem   `json:"action_items"`
	CreatedAt             time.Time      `json:"created_at"`
}

type FiveWhyItem struct {
	Why    string `json:"why"`
	Answer string `json:"answer"`
}

type ActionItem struct {
	ID                string     `json:"id"`
	RCAID             string     `json:"rca_id"`
	Description       string     `json:"description"`
	Assignee          string     `json:"assignee"`
	DueDate           *time.Time `json:"due_date,omitempty"`
	Status            string     `json:"status"` // open, in_progress, completed
	ExternalTicketURL string     `json:"external_ticket_url,omitempty"`
}

// GCP Cloud Monitoring Webhook Payload Structure
// https://cloud.google.com/monitoring/support/notification-options#webhooks
type GCPMonitoringPayload struct {
	Version  string      `json:"version"`
	Incident GCPIncident `json:"incident"`
}

type GCPIncident struct {
	IncidentID    string            `json:"incident_id"`
	ResourceID    string            `json:"resource_id"`
	ResourceName  string            `json:"resource_name"`
	ResourceType  string            `json:"resource_type"`
	Resource      GCPResource       `json:"resource"`
	Metric        GCPMetric         `json:"metric"`
	StartedAt     int64             `json:"started_at"` // Unix timestamp in seconds
	EndedAt       int64             `json:"ended_at"`
	State         string            `json:"state"` // "open" or "closed"
	Summary       string            `json:"summary"`
	ConditionName string            `json:"condition_name"`
	Condition     GCPCondition      `json:"condition"`
	PolicyName    string            `json:"policy_name"`
	URL           string            `json:"url"` // Cloud Console incident link
	Documentation *GCPDocumentation `json:"documentation,omitempty"`
}

type GCPResource struct {
	Type   string            `json:"type"` // e.g. "cloud_run_revision", "gke_container", "gce_instance"
	Labels map[string]string `json:"labels"`
}

type GCPMetric struct {
	Type        string            `json:"type"` // e.g. "run.googleapis.com/container/memory/utilizations"
	DisplayName string            `json:"displayName"`
	Labels      map[string]string `json:"labels"`
}

type GCPCondition struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type GCPDocumentation struct {
	Content  string `json:"content"`
	MimeType string `json:"mime_type"`
}

// GCP Cloud Logging Export Entry / Log Sink Webhook Payload
type GCPLogEntry struct {
	LogName      string                 `json:"logName"`
	Resource     GCPResource            `json:"resource"`
	Timestamp    time.Time              `json:"timestamp"`
	Severity     string                 `json:"severity"` // DEFAULT, DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL, ALERT, EMERGENCY
	TextPayload  string                 `json:"textPayload,omitempty"`
	JSONPayload  map[string]interface{} `json:"jsonPayload,omitempty"`
	InsertID     string                 `json:"insertId"`
	Trace        string                 `json:"trace,omitempty"`
	SpanID       string                 `json:"spanId,omitempty"`
}

// AICopilotTriageResult represents structured AI triage output
type AICopilotTriageResult struct {
	IncidentID          string                `json:"incident_id"`
	Summary             string                `json:"summary"`
	ProbableRootCause   string                `json:"probable_root_cause"`
	RecommendedActions  []RecommendedAction   `json:"recommended_actions"`
	SimilarIncidents    []SimilarIncidentInfo `json:"similar_incidents"`
}

type RecommendedAction struct {
	Step        int    `json:"step"`
	Action      string `json:"action"`
	Command     string `json:"command,omitempty"`
	RunbookLink string `json:"runbook_link,omitempty"`
}

type SimilarIncidentInfo struct {
	IncidentID      string  `json:"incident_id"`
	IncidentNumber  int     `json:"incident_number"`
	Title           string  `json:"title"`
	SimilarityScore float64 `json:"similarity_score"`
	Resolution      string  `json:"resolution"`
}
