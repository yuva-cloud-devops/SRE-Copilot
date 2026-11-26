package repository

import (
	"sre-copilot/internal/models"
)

type IncidentRepository interface {
	ListIncidents(status models.IncidentStatus, severity models.IncidentSeverity) ([]models.Incident, error)
	GetIncident(id string) (*models.Incident, error)
	CreateIncident(inc *models.Incident) error
	UpdateIncident(inc *models.Incident) error
	AddTimelineEvent(event *models.TimelineEvent) error
	GetTimeline(incidentID string) ([]models.TimelineEvent, error)
	AddAlert(alert *models.Alert) error
	GetAlerts(incidentID string) ([]models.Alert, error)
	AddLog(record *models.GCPLogRecord) error
	GetLogs(incidentID string) ([]models.GCPLogRecord, error)
	FindOpenIncidentForAlert(serviceName, env string, windowMinutes int) (*models.Incident, error)
}

type KnowledgeRepository interface {
	Search(query string, serviceName string, limit int) ([]models.KnowledgeDocument, error)
	ListAll() ([]models.KnowledgeDocument, error)
	GetDocument(id string) (*models.KnowledgeDocument, error)
	AddDocument(doc *models.KnowledgeDocument) error
}

type RCARepository interface {
	GetRCA(incidentID string) (*models.RCAReport, error)
	SaveRCA(rca *models.RCAReport) error
	ListRCAs() ([]models.RCAReport, error)
}
