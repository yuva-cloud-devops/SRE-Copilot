package services

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"sre-copilot/internal/models"
	"sre-copilot/internal/repository"
)

type AlertCorrelator struct {
	repo        repository.IncidentRepository
	broadcaster *SSEBroadcaster
	mu          sync.Mutex
	// Fingerprint cache for deduplication: fingerprint -> alertID
	recentFingerprints map[string]*models.Alert
	windowMinutes      int
}

func NewAlertCorrelator(repo repository.IncidentRepository, broadcaster *SSEBroadcaster) *AlertCorrelator {
	return &AlertCorrelator{
		repo:               repo,
		broadcaster:        broadcaster,
		recentFingerprints: make(map[string]*models.Alert),
		windowMinutes:      15,
	}
}

// ComputeFingerprint creates a deterministic MD5 hash for alert deduplication
func ComputeFingerprint(serviceName, alertName, environment string) string {
	raw := fmt.Sprintf("%s|%s|%s", strings.ToLower(serviceName), strings.ToLower(alertName), strings.ToLower(environment))
	hash := md5.Sum([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// IngestAlert processes incoming alert, deduplicates, and correlates to existing or new incident
func (c *AlertCorrelator) IngestAlert(alert *models.Alert) (*models.Incident, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if alert.Fingerprint == "" {
		alert.Fingerprint = ComputeFingerprint(alert.ServiceName, alert.AlertName, alert.Environment)
	}

	now := time.Now()
	alert.LastSeenAt = now

	// 1. Deduplication check within sliding window
	if existing, found := c.recentFingerprints[alert.Fingerprint]; found {
		if now.Sub(existing.LastSeenAt) < time.Duration(c.windowMinutes)*time.Minute {
			existing.RecurrenceCount++
			existing.LastSeenAt = now
			// Deduplicated/suppressed
			return nil, true, nil
		}
	}

	// 2. Correlation check: Find active incident matching same service and environment within window
	incident, err := c.repo.FindOpenIncidentForAlert(alert.ServiceName, alert.Environment, c.windowMinutes)
	if err != nil {
		return nil, false, err
	}

	if incident == nil {
		// 3. Create new incident
		incID := uuid.New().String()

		// Elevate severity if alert is critical
		sev := alert.Severity
		if sev == "" {
			sev = models.SeverityP3
		}

		incident = &models.Incident{
			ID:             incID,
			Title:          fmt.Sprintf("Outage on %s: %s", alert.ServiceName, alert.AlertName),
			Summary:        alert.Description,
			ServiceName:    alert.ServiceName,
			Environment:    alert.Environment,
			Severity:       sev,
			Status:         models.StatusInvestigating,
			LeadResponder:  "Unassigned",
			StartedAt:      now,
			CreatedAt:      now,
			UpdatedAt:      now,
			AlertCount:     1,
		}

		if err := c.repo.CreateIncident(incident); err != nil {
			return nil, false, err
		}

		// Initial timeline event
		_ = c.repo.AddTimelineEvent(&models.TimelineEvent{
			ID:          uuid.New().String(),
			IncidentID:  incident.ID,
			EventType:   models.EventAlertTriggered,
			Description: fmt.Sprintf("Incident auto-created from alert '%s' (%s) on %s [%s]", alert.AlertName, alert.Severity, alert.ServiceName, alert.Environment),
			CreatedAt:   now,
		})

		c.broadcaster.Broadcast("new_incident", fmt.Sprintf(`{"id":"%s","title":"%s"}`, incident.ID, incident.Title))
	} else {
		// Existing incident found: correlate & link
		incident.AlertCount++
		incident.UpdatedAt = now
		_ = c.repo.UpdateIncident(incident)

		_ = c.repo.AddTimelineEvent(&models.TimelineEvent{
			ID:          uuid.New().String(),
			IncidentID:  incident.ID,
			EventType:   models.EventAlertTriggered,
			Description: fmt.Sprintf("Correlated alert '%s' (%s) merged into incident.", alert.AlertName, alert.Severity),
			CreatedAt:   now,
		})

		c.broadcaster.Broadcast("timeline_update", fmt.Sprintf(`{"incident_id":"%s","description":"Correlated alert %s"}`, incident.ID, alert.AlertName))
	}

	// Link alert to incident
	if alert.ID == "" {
		alert.ID = uuid.New().String()
	}
	alert.IncidentID = incident.ID
	alert.RecurrenceCount = 1
	if alert.TriggeredAt.IsZero() {
		alert.TriggeredAt = now
	}

	_ = c.repo.AddAlert(alert)
	c.recentFingerprints[alert.Fingerprint] = alert

	return incident, false, nil
}

// IngestGCPAlert processes Google Cloud Monitoring alerting policy webhook notifications
func (c *AlertCorrelator) IngestGCPAlert(payload *models.GCPMonitoringPayload) (*models.Incident, bool, error) {
	gcpInc := payload.Incident

	// Extract service name from resource labels
	serviceName := gcpInc.Resource.Labels["service_name"]
	if serviceName == "" {
		serviceName = gcpInc.Resource.Labels["container_name"]
	}
	if serviceName == "" {
		serviceName = gcpInc.Resource.Labels["instance_id"]
	}
	if serviceName == "" {
		serviceName = gcpInc.ResourceName
	}
	if serviceName == "" {
		serviceName = "gcp-cloud-run-service"
	}

	env := gcpInc.Resource.Labels["environment"]
	if env == "" {
		env = "production"
	}

	// Derive severity from policy name or condition
	severity := models.SeverityP2
	pLower := strings.ToLower(gcpInc.PolicyName + " " + gcpInc.ConditionName + " " + gcpInc.Summary)
	if strings.Contains(pLower, "critical") || strings.Contains(pLower, "outage") || strings.Contains(pLower, "exhaustion") || strings.Contains(pLower, "down") {
		severity = models.SeverityP1
	} else if strings.Contains(pLower, "warning") || strings.Contains(pLower, "latency") {
		severity = models.SeverityP3
	}

	alertName := gcpInc.PolicyName
	if alertName == "" {
		alertName = gcpInc.ConditionName
	}
	if alertName == "" {
		alertName = "GCP Alert: " + gcpInc.Summary
	}

	description := gcpInc.Summary
	if gcpInc.Documentation != nil && gcpInc.Documentation.Content != "" {
		description = fmt.Sprintf("%s\nRunbook: %s", description, gcpInc.Documentation.Content)
	}

	triggeredAt := time.Now()
	if gcpInc.StartedAt > 0 {
		triggeredAt = time.Unix(gcpInc.StartedAt, 0)
	}

	labels := make(map[string]string)
	for k, v := range gcpInc.Resource.Labels {
		labels[k] = v
	}
	labels["gcp_resource_type"] = gcpInc.Resource.Type
	labels["gcp_metric"] = gcpInc.Metric.Type

	alert := &models.Alert{
		AlertName:       alertName,
		Source:          "gcp_monitoring",
		ServiceName:     serviceName,
		Environment:     env,
		Severity:        severity,
		Status:          models.AlertStatusFiring,
		Description:     description,
		GCPPolicyName:   gcpInc.PolicyName,
		GCPCondition:    gcpInc.ConditionName,
		GCPMetricType:   gcpInc.Metric.Type,
		GCPConsoleURL:   gcpInc.URL,
		Labels:          labels,
		Annotations:     map[string]string{"incident_url": gcpInc.URL, "summary": gcpInc.Summary},
		TriggeredAt:     triggeredAt,
	}

	return c.IngestAlert(alert)
}

// IngestGCPLog processes exported Google Cloud Logging error entries and correlates them to active incidents
func (c *AlertCorrelator) IngestGCPLog(entry *models.GCPLogEntry) (*models.Incident, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Extract service name
	serviceName := entry.Resource.Labels["service_name"]
	if serviceName == "" {
		serviceName = entry.Resource.Labels["container_name"]
	}
	if serviceName == "" {
		serviceName = entry.Resource.Labels["instance_id"]
	}
	if serviceName == "" {
		serviceName = "payment-service"
	}

	env := entry.Resource.Labels["environment"]
	if env == "" {
		env = "production"
	}

	message := entry.TextPayload
	if message == "" && entry.JSONPayload != nil {
		if msg, ok := entry.JSONPayload["message"].(string); ok {
			message = msg
		} else if errVal, ok := entry.JSONPayload["error"].(string); ok {
			message = errVal
		} else {
			message = fmt.Sprintf("%v", entry.JSONPayload)
		}
	}
	if message == "" {
		message = "Anomalous GCP error log detected in " + entry.LogName
	}

	// 1. Check for active incident for this service within window
	incident, err := c.repo.FindOpenIncidentForAlert(serviceName, env, c.windowMinutes)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	timestamp := entry.Timestamp
	if timestamp.IsZero() {
		timestamp = now
	}

	// 2. If no open incident exists and log severity is CRITICAL or EMERGENCY, auto-create incident
	sevUpper := strings.ToUpper(entry.Severity)
	if incident == nil {
		if sevUpper == "CRITICAL" || sevUpper == "EMERGENCY" || sevUpper == "ALERT" {
			incID := uuid.New().String()
			incident = &models.Incident{
				ID:             incID,
				Title:          fmt.Sprintf("Critical GCP Log Anomaly on %s: %s", serviceName, entry.Severity),
				Summary:        message,
				ServiceName:    serviceName,
				Environment:    env,
				Severity:       models.SeverityP1,
				Status:         models.StatusInvestigating,
				LeadResponder:  "Unassigned",
				StartedAt:      timestamp,
				CreatedAt:      now,
				UpdatedAt:      now,
				AlertCount:     1,
			}
			if err := c.repo.CreateIncident(incident); err != nil {
				return nil, err
			}
			c.broadcaster.Broadcast("new_incident", fmt.Sprintf(`{"id":"%s","title":"%s"}`, incident.ID, incident.Title))
		} else {
			// Normal error with no active incident
			return nil, nil
		}
	}

	// 3. Attach log record to incident
	logRecord := &models.GCPLogRecord{
		ID:          uuid.New().String(),
		IncidentID:  incident.ID,
		LogName:     entry.LogName,
		Severity:    entry.Severity,
		ServiceName: serviceName,
		Message:     message,
		Payload:     entry.JSONPayload,
		Timestamp:   timestamp,
	}

	if err := c.repo.AddLog(logRecord); err != nil {
		return incident, err
	}

	// 4. Add timeline event for log correlation
	displayMsg := message
	if len(displayMsg) > 140 {
		displayMsg = displayMsg[:137] + "..."
	}
	_ = c.repo.AddTimelineEvent(&models.TimelineEvent{
		ID:          uuid.New().String(),
		IncidentID:  incident.ID,
		EventType:   models.EventLogCorrelated,
		Description: fmt.Sprintf("GCP Cloud Logging [%s]: %s", entry.Severity, displayMsg),
		CreatedAt:   now,
	})

	c.broadcaster.Broadcast("timeline_update", fmt.Sprintf(`{"incident_id":"%s","description":"Correlated GCP Log [%s]"}`, incident.ID, entry.Severity))

	return incident, nil
}
