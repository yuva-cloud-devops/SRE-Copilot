package handlers

import (
	"log"
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
		log.Printf("❌ Failed to parse GCP Monitoring webhook: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GCP Cloud Monitoring payload: " + err.Error()})
		return
	}

	if payload.Incident.IncidentID == "" && payload.Incident.Summary == "" {
		log.Printf("❌ Malformed GCP incident data in webhook payload: %+v", payload)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Malformed GCP incident data in webhook payload"})
		return
	}

	log.Printf("📥 Ingesting GCP Monitoring Alert: policy='%s', summary='%s', id='%s'", payload.Incident.PolicyName, payload.Incident.Summary, payload.Incident.IncidentID)

	incident, isDuplicate, err := h.correlator.IngestGCPAlert(&payload)
	if err != nil {
		log.Printf("❌ Correlator error on GCP Monitoring alert: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process GCP alert: " + err.Error()})
		return
	}

	if isDuplicate {
		log.Printf("ℹ️ Alert suppressed (duplicate within sliding window)")
	} else if incident != nil {
		log.Printf("🚨 Correlated into Incident #%d [%s]: %s (Severity: %s)", incident.IncidentNumber, incident.ID, incident.Title, incident.Severity)
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
		log.Printf("❌ Failed to parse GCP Logging webhook: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GCP Cloud Logging payload: " + err.Error()})
		return
	}

	log.Printf("📥 Ingesting GCP Log Event: logName='%s', severity='%s'", entry.LogName, entry.Severity)

	incident, err := h.correlator.IngestGCPLog(&entry)
	if err != nil {
		log.Printf("❌ Correlator error on GCP Log ingestion: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ingest GCP log: " + err.Error()})
		return
	}

	if incident != nil {
		log.Printf("🔗 Log correlated into active Incident #%d [%s]", incident.IncidentNumber, incident.ID)
	} else {
		log.Printf("ℹ️ Log received, no open incident to correlate")
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
