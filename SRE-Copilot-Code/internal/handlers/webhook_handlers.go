package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sre-copilot/internal/models"
	"sre-copilot/internal/services"
)

type WebhookHandler struct {
	correlator *services.AlertCorrelator
}

func NewWebhookHandler(correlator *services.AlertCorrelator) *WebhookHandler {
	return &WebhookHandler{correlator: correlator}
}

// GCPMonitoringWebhook handles Google Cloud Monitoring alerting policy webhook payloads
// Docs: https://cloud.google.com/monitoring/support/notification-options#webhooks
func (h *WebhookHandler) GCPMonitoringWebhook(c *gin.Context) {
	var payload models.GCPMonitoringPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GCP Cloud Monitoring payload: " + err.Error()})
		return
	}

	if payload.Incident.IncidentID == "" && payload.Incident.Summary == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Malformed GCP incident data in webhook payload"})
		return
	}

	incident, isDuplicate, err := h.correlator.IngestGCPAlert(&payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process GCP alert: " + err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":          "accepted",
		"provider":        "google_cloud_monitoring",
		"gcp_incident_id": payload.Incident.IncidentID,
		"policy_name":     payload.Incident.PolicyName,
		"suppressed":      isDuplicate,
		"copilot_incident": func() interface{} {
			if incident != nil {
				return gin.H{"id": incident.ID, "title": incident.Title, "severity": incident.Severity}
			}
			return nil
		}(),
	})
}

// GCPLoggingWebhook handles log export sinks or log-based alerts from Google Cloud Logging
func (h *WebhookHandler) GCPLoggingWebhook(c *gin.Context) {
	var entry models.GCPLogEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GCP Cloud Logging payload: " + err.Error()})
		return
	}

	incident, err := h.correlator.IngestGCPLog(&entry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ingest GCP log: " + err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":      "accepted",
		"provider":    "google_cloud_logging",
		"severity":    entry.Severity,
		"log_name":    entry.LogName,
		"correlated":  incident != nil,
		"incident_id": func() string {
			if incident != nil {
				return incident.ID
			}
			return ""
		}(),
	})
}
