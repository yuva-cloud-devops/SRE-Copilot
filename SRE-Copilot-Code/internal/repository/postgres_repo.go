package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"sre-copilot/internal/models"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ListIncidents(status models.IncidentStatus, severity models.IncidentSeverity) ([]models.Incident, error) {
	query := `
		SELECT id, incident_number, title, summary, service_name, environment,
		       severity, status, lead_responder, root_cause_hypothesis, started_at,
		       resolved_at, created_at, updated_at, alert_count
		FROM incidents
		WHERE 1=1
	`
	var args []interface{}
	idx := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, string(status))
		idx++
	}
	if severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", idx)
		args = append(args, string(severity))
		idx++
	}

	query += " ORDER BY started_at DESC LIMIT 50"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []models.Incident
	for rows.Next() {
		var inc models.Incident
		var summary, leadResponder, hypothesis sql.NullString
		var resolvedAt sql.NullTime

		err := rows.Scan(
			&inc.ID, &inc.IncidentNumber, &inc.Title, &summary, &inc.ServiceName,
			&inc.Environment, &inc.Severity, &inc.Status, &leadResponder,
			&hypothesis, &inc.StartedAt, &resolvedAt, &inc.CreatedAt,
			&inc.UpdatedAt, &inc.AlertCount,
		)
		if err != nil {
			return nil, err
		}

		if summary.Valid {
			inc.Summary = summary.String
		}
		if leadResponder.Valid {
			inc.LeadResponder = leadResponder.String
		}
		if hypothesis.Valid {
			inc.RootCauseHypothesis = hypothesis.String
		}
		if resolvedAt.Valid {
			inc.ResolvedAt = &resolvedAt.Time
		}

		incidents = append(incidents, inc)
	}

	return incidents, nil
}

func (r *PostgresRepository) GetIncident(id string) (*models.Incident, error) {
	query := `
		SELECT id, incident_number, title, summary, service_name, environment,
		       severity, status, lead_responder, root_cause_hypothesis, started_at,
		       resolved_at, created_at, updated_at, alert_count
		FROM incidents
		WHERE id::text = $1
	`
	var inc models.Incident
	var summary, leadResponder, hypothesis sql.NullString
	var resolvedAt sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&inc.ID, &inc.IncidentNumber, &inc.Title, &summary, &inc.ServiceName,
		&inc.Environment, &inc.Severity, &inc.Status, &leadResponder,
		&hypothesis, &inc.StartedAt, &resolvedAt, &inc.CreatedAt,
		&inc.UpdatedAt, &inc.AlertCount,
	)
	if err != nil {
		return nil, fmt.Errorf("incident not found: %w", err)
	}

	if summary.Valid {
		inc.Summary = summary.String
	}
	if leadResponder.Valid {
		inc.LeadResponder = leadResponder.String
	}
	if hypothesis.Valid {
		inc.RootCauseHypothesis = hypothesis.String
	}
	if resolvedAt.Valid {
		inc.ResolvedAt = &resolvedAt.Time
	}

	// Fetch alerts
	alerts, _ := r.GetAlerts(id)
	inc.Alerts = alerts
	inc.AlertCount = len(alerts)

	// Fetch timeline
	timeline, _ := r.GetTimeline(id)
	inc.Timeline = timeline

	// Fetch correlated logs
	logs, _ := r.GetLogs(id)
	inc.Logs = logs

	return &inc, nil
}

func (r *PostgresRepository) CreateIncident(inc *models.Incident) error {
	query := `
		INSERT INTO incidents (id, title, summary, service_name, environment, severity, status, lead_responder, started_at, created_at, updated_at, alert_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING incident_number
	`
	now := time.Now()
	if inc.CreatedAt.IsZero() {
		inc.CreatedAt = now
	}
	if inc.UpdatedAt.IsZero() {
		inc.UpdatedAt = now
	}
	if inc.StartedAt.IsZero() {
		inc.StartedAt = now
	}

	return r.db.QueryRow(
		query, inc.ID, inc.Title, inc.Summary, inc.ServiceName, inc.Environment,
		string(inc.Severity), string(inc.Status), inc.LeadResponder,
		inc.StartedAt, inc.CreatedAt, inc.UpdatedAt, inc.AlertCount,
	).Scan(&inc.IncidentNumber)
}

func (r *PostgresRepository) UpdateIncident(inc *models.Incident) error {
	query := `
		UPDATE incidents
		SET title = $1, summary = $2, severity = $3, status = $4,
		    lead_responder = $5, root_cause_hypothesis = $6,
		    resolved_at = $7, updated_at = NOW(), alert_count = $8
		WHERE id::text = $9
	`
	_, err := r.db.Exec(
		query, inc.Title, inc.Summary, string(inc.Severity), string(inc.Status),
		inc.LeadResponder, inc.RootCauseHypothesis, inc.ResolvedAt, inc.AlertCount, inc.ID,
	)
	return err
}

func (r *PostgresRepository) AddTimelineEvent(event *models.TimelineEvent) error {
	query := `
		INSERT INTO incident_timeline (id, incident_id, event_type, description, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	_, err := r.db.Exec(query, event.ID, event.IncidentID, string(event.EventType), event.Description, event.CreatedBy, event.CreatedAt)
	return err
}

func (r *PostgresRepository) GetTimeline(incidentID string) ([]models.TimelineEvent, error) {
	query := `
		SELECT id, incident_id, event_type, description, created_by, created_at
		FROM incident_timeline
		WHERE incident_id::text = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(query, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.TimelineEvent
	for rows.Next() {
		var e models.TimelineEvent
		var createdBy sql.NullString
		err := rows.Scan(&e.ID, &e.IncidentID, &e.EventType, &e.Description, &createdBy, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		if createdBy.Valid {
			e.CreatedBy = createdBy.String
		}
		events = append(events, e)
	}
	return events, nil
}

func (r *PostgresRepository) AddAlert(alert *models.Alert) error {
	labelsJSON, _ := json.Marshal(alert.Labels)
	annotationsJSON, _ := json.Marshal(alert.Annotations)

	query := `
		INSERT INTO alerts (
			id, incident_id, fingerprint, alert_name, source, service_name,
			environment, severity, status, description, gcp_policy_name,
			gcp_condition, gcp_metric_type, gcp_console_url, labels, annotations,
			recurrence_count, triggered_at, last_seen_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	_, err := r.db.Exec(
		query, alert.ID, alert.IncidentID, alert.Fingerprint, alert.AlertName,
		alert.Source, alert.ServiceName, alert.Environment, string(alert.Severity),
		string(alert.Status), alert.Description, alert.GCPPolicyName, alert.GCPCondition,
		alert.GCPMetricType, alert.GCPConsoleURL, string(labelsJSON), string(annotationsJSON),
		alert.RecurrenceCount, alert.TriggeredAt, alert.LastSeenAt,
	)
	return err
}

func (r *PostgresRepository) GetAlerts(incidentID string) ([]models.Alert, error) {
	query := `
		SELECT id, incident_id, fingerprint, alert_name, source, service_name,
		       environment, severity, status, description, gcp_policy_name,
		       gcp_condition, gcp_metric_type, gcp_console_url, labels,
		       recurrence_count, triggered_at, last_seen_at
		FROM alerts
		WHERE incident_id::text = $1
		ORDER BY triggered_at ASC
	`
	rows, err := r.db.Query(query, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var a models.Alert
		var labelsRaw []byte
		var policy, cond, metric, url sql.NullString

		err := rows.Scan(
			&a.ID, &a.IncidentID, &a.Fingerprint, &a.AlertName, &a.Source,
			&a.ServiceName, &a.Environment, &a.Severity, &a.Status, &a.Description,
			&policy, &cond, &metric, &url, &labelsRaw,
			&a.RecurrenceCount, &a.TriggeredAt, &a.LastSeenAt,
		)
		if err != nil {
			return nil, err
		}

		if policy.Valid {
			a.GCPPolicyName = policy.String
		}
		if cond.Valid {
			a.GCPCondition = cond.String
		}
		if metric.Valid {
			a.GCPMetricType = metric.String
		}
		if url.Valid {
			a.GCPConsoleURL = url.String
		}
		if len(labelsRaw) > 0 {
			_ = json.Unmarshal(labelsRaw, &a.Labels)
		}

		alerts = append(alerts, a)
	}
	return alerts, nil
}

func (r *PostgresRepository) AddLog(record *models.GCPLogRecord) error {
	payloadJSON, _ := json.Marshal(record.Payload)
	query := `
		INSERT INTO gcp_logs (id, incident_id, log_name, severity, service_name, message, payload, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(query, record.ID, record.IncidentID, record.LogName, record.Severity, record.ServiceName, record.Message, string(payloadJSON), record.Timestamp)
	return err
}

func (r *PostgresRepository) GetLogs(incidentID string) ([]models.GCPLogRecord, error) {
	query := `
		SELECT id, incident_id, log_name, severity, service_name, message, timestamp
		FROM gcp_logs
		WHERE incident_id::text = $1
		ORDER BY timestamp DESC
	`
	rows, err := r.db.Query(query, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.GCPLogRecord
	for rows.Next() {
		var l models.GCPLogRecord
		err := rows.Scan(&l.ID, &l.IncidentID, &l.LogName, &l.Severity, &l.ServiceName, &l.Message, &l.Timestamp)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (r *PostgresRepository) FindOpenIncidentForAlert(serviceName, env string, windowMinutes int) (*models.Incident, error) {
	query := `
		SELECT id, incident_number, title, summary, service_name, environment,
		       severity, status, lead_responder, started_at, updated_at, alert_count
		FROM incidents
		WHERE status != 'resolved'
		  AND LOWER(service_name) = LOWER($1)
		  AND LOWER(environment) = LOWER($2)
		  AND updated_at >= NOW() - ($3 * INTERVAL '1 minute')
		ORDER BY updated_at DESC
		LIMIT 1
	`
	var inc models.Incident
	var summary, leadResponder sql.NullString

	err := r.db.QueryRow(query, serviceName, env, windowMinutes).Scan(
		&inc.ID, &inc.IncidentNumber, &inc.Title, &summary, &inc.ServiceName,
		&inc.Environment, &inc.Severity, &inc.Status, &leadResponder,
		&inc.StartedAt, &inc.UpdatedAt, &inc.AlertCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if summary.Valid {
		inc.Summary = summary.String
	}
	if leadResponder.Valid {
		inc.LeadResponder = leadResponder.String
	}

	return &inc, nil
}

func (r *PostgresRepository) Search(query string, serviceName string, limit int) ([]models.KnowledgeDocument, error) {
	sqlQuery := `
		SELECT id, title, document_type, service_name, content, tags, created_at
		FROM knowledge_documents
		WHERE 1=1
	`
	var args []interface{}
	idx := 1

	if serviceName != "" {
		sqlQuery += fmt.Sprintf(" AND LOWER(service_name) = LOWER($%d)", idx)
		args = append(args, serviceName)
		idx++
	}

	if query != "" {
		sqlQuery += fmt.Sprintf(" AND (LOWER(title) LIKE $%d OR LOWER(content) LIKE $%d)", idx, idx)
		args = append(args, "%"+strings.ToLower(query)+"%")
		idx++
	}

	sqlQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", idx)
	args = append(args, limit)

	rows, err := r.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.KnowledgeDocument
	for rows.Next() {
		var doc models.KnowledgeDocument
		var service sql.NullString
		err := rows.Scan(&doc.ID, &doc.Title, &doc.DocumentType, &service, &doc.Content, pq.Array(&doc.Tags), &doc.CreatedAt)
		if err != nil {
			return nil, err
		}
		if service.Valid {
			doc.ServiceName = service.String
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

func (r *PostgresRepository) ListAll() ([]models.KnowledgeDocument, error) {
	return r.Search("", "", 50)
}

func (r *PostgresRepository) GetDocument(id string) (*models.KnowledgeDocument, error) {
	query := `
		SELECT id, title, document_type, service_name, content, tags, created_at
		FROM knowledge_documents
		WHERE id::text = $1
	`
	var doc models.KnowledgeDocument
	var service sql.NullString
	err := r.db.QueryRow(query, id).Scan(&doc.ID, &doc.Title, &doc.DocumentType, &service, &doc.Content, pq.Array(&doc.Tags), &doc.CreatedAt)
	if err != nil {
		return nil, err
	}
	if service.Valid {
		doc.ServiceName = service.String
	}
	return &doc, nil
}

func (r *PostgresRepository) AddDocument(doc *models.KnowledgeDocument) error {
	query := `
		INSERT INTO knowledge_documents (id, title, document_type, service_name, content, tags, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	_, err := r.db.Exec(query, doc.ID, doc.Title, doc.DocumentType, doc.ServiceName, doc.Content, pq.Array(doc.Tags), doc.CreatedAt)
	return err
}

func (r *PostgresRepository) GetRCA(incidentID string) (*models.RCAReport, error) {
	query := `
		SELECT id, incident_id, incident_number, title, executive_summary, impact_summary,
		       root_cause_analysis, five_whys, preventive_actions, published, author,
		       action_items, created_at
		FROM rca_reports
		WHERE incident_id::text = $1
	`
	var rca models.RCAReport
	var fiveWhysRaw, prevActionsRaw, actionItemsRaw []byte
	var author sql.NullString

	err := r.db.QueryRow(query, incidentID).Scan(
		&rca.ID, &rca.IncidentID, &rca.IncidentNumber, &rca.Title,
		&rca.ExecutiveSummary, &rca.ImpactSummary, &rca.RootCauseAnalysis,
		&fiveWhysRaw, &prevActionsRaw, &rca.Published, &author,
		&actionItemsRaw, &rca.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if author.Valid {
		rca.Author = author.String
	}
	if len(fiveWhysRaw) > 0 {
		_ = json.Unmarshal(fiveWhysRaw, &rca.FiveWhys)
	}
	if len(prevActionsRaw) > 0 {
		_ = json.Unmarshal(prevActionsRaw, &rca.PreventiveActions)
	}
	if len(actionItemsRaw) > 0 {
		_ = json.Unmarshal(actionItemsRaw, &rca.ActionItems)
	}

	return &rca, nil
}

func (r *PostgresRepository) SaveRCA(rca *models.RCAReport) error {
	fiveWhysJSON, _ := json.Marshal(rca.FiveWhys)
	prevActionsJSON, _ := json.Marshal(rca.PreventiveActions)
	actionItemsJSON, _ := json.Marshal(rca.ActionItems)

	query := `
		INSERT INTO rca_reports (
			id, incident_id, incident_number, title, executive_summary, impact_summary,
			root_cause_analysis, five_whys, preventive_actions, published, author, action_items, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (incident_id) DO UPDATE
		SET title = EXCLUDED.title, executive_summary = EXCLUDED.executive_summary,
		    impact_summary = EXCLUDED.impact_summary, root_cause_analysis = EXCLUDED.root_cause_analysis,
		    five_whys = EXCLUDED.five_whys, preventive_actions = EXCLUDED.preventive_actions,
		    published = EXCLUDED.published, action_items = EXCLUDED.action_items
	`
	_, err := r.db.Exec(
		query, rca.ID, rca.IncidentID, rca.IncidentNumber, rca.Title,
		rca.ExecutiveSummary, rca.ImpactSummary, rca.RootCauseAnalysis,
		string(fiveWhysJSON), string(prevActionsJSON), rca.Published, rca.Author, string(actionItemsJSON),
	)
	return err
}

func (r *PostgresRepository) ListRCAs() ([]models.RCAReport, error) {
	query := `
		SELECT id, incident_id, incident_number, title, executive_summary, impact_summary,
		       root_cause_analysis, published, author, created_at
		FROM rca_reports
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rcas []models.RCAReport
	for rows.Next() {
		var rca models.RCAReport
		var author sql.NullString
		err := rows.Scan(
			&rca.ID, &rca.IncidentID, &rca.IncidentNumber, &rca.Title,
			&rca.ExecutiveSummary, &rca.ImpactSummary, &rca.RootCauseAnalysis,
			&rca.Published, &author, &rca.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if author.Valid {
			rca.Author = author.String
		}
		rcas = append(rcas, rca)
	}
	return rcas, nil
}
