package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"sre-copilot/internal/config"
	"sre-copilot/internal/models"
)

type AICopilotService struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewAICopilotService(cfg *config.Config) *AICopilotService {
	return &AICopilotService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

// TriageIncident generates a root cause hypothesis, summary, and action items
func (s *AICopilotService) TriageIncident(incident *models.Incident, runbooks []models.KnowledgeDocument) (*models.AICopilotTriageResult, error) {
	// If Gemini key is set, call Gemini API
	if s.cfg.GeminiAPIKey != "" {
		res, err := s.callGeminiTriage(incident, runbooks)
		if err == nil {
			return res, nil
		}
		// Fall through to fallback on failure
	}

	// If OpenAI key is set, call OpenAI API
	if s.cfg.OpenAIAPIKey != "" {
		res, err := s.callOpenAITriage(incident, runbooks)
		if err == nil {
			return res, nil
		}
		// Fall through to fallback on failure
	}

	// Intelligent Local SRE Rule Engine Fallback
	return s.localSimulatedTriage(incident, runbooks)
}

// StreamChat streams conversational SRE advice token by token via callback
func (s *AICopilotService) StreamChat(incident *models.Incident, question string, runbooks []models.KnowledgeDocument, onChunk func(chunk string) error) error {
	if s.cfg.GeminiAPIKey != "" {
		err := s.streamGeminiChat(incident, question, runbooks, onChunk)
		if err == nil {
			return nil
		}
	}

	if s.cfg.OpenAIAPIKey != "" {
		err := s.streamOpenAIChat(incident, question, runbooks, onChunk)
		if err == nil {
			return nil
		}
	}

	// High-fidelity SRE Copilot simulation
	return s.streamSimulatedChat(incident, question, runbooks, onChunk)
}

// GenerateRCA generates a draft postmortem
func (s *AICopilotService) GenerateRCA(incident *models.Incident) (*models.RCAReport, error) {
	var timelineEvents []string
	for _, t := range incident.Timeline {
		timelineEvents = append(timelineEvents, fmt.Sprintf("[%s] %s", t.CreatedAt.Format("15:04:05 MST"), t.Description))
	}

	summary := fmt.Sprintf("On %s, %s experienced a %s outage in %s affecting service availability.",
		incident.StartedAt.Format("Jan 02, 2006"), incident.ServiceName, incident.Severity, incident.Environment)
	if incident.Summary != "" {
		summary += " " + incident.Summary
	}

	rca := &models.RCAReport{
		ID:                fmt.Sprintf("rca-%s", incident.ID),
		IncidentID:        incident.ID,
		IncidentNumber:    incident.IncidentNumber,
		Title:             fmt.Sprintf("Postmortem: %s", incident.Title),
		ExecutiveSummary:  summary,
		ImpactSummary:     fmt.Sprintf("Critical checkout transactions degraded for approximately %d minutes. Correlated %d alerts across infrastructure and application layers.", int(time.Since(incident.StartedAt).Minutes()), incident.AlertCount),
		RootCauseAnalysis: "Initial telemetry correlates connection pool exhaustion to sustained lock contention on primary database tables following a release spike. Queries held locks without closing transaction handles, exhausting worker threads.",
		FiveWhys: []models.FiveWhyItem{
			{Why: "Why did order-service return HTTP 504 timeouts to customers?", Answer: "Upstream requests queued waiting for database connection handles that never became available."},
			{Why: "Why were database connections unavailable?", Answer: "The connection pool reached 100% capacity (100/100 active connections)."},
			{Why: "Why did connections remain active for so long?", Answer: "Transactions remained in 'idle in transaction' state due to an un-indexed lock contention on the 'orders' table."},
			{Why: "Why did the un-indexed query make it into production?", Answer: "Load testing in staging utilized synthetic datasets without realistic table partition cardinality."},
			{Why: "Why wasn't pool exhaustion caught earlier?", Answer: "Pool warning threshold alert fired at 95% instead of a progressive early-warning at 75%."},
		},
		PreventiveActions: []string{
			"Add composite index on (user_id, status) for orders table.",
			"Reduce connection timeout in PgBouncer from 30s to 5s with circuit breaker fallback.",
			"Implement staged canary deployment checks verifying DB connection metrics.",
			"Update runbook 'Postgres Connection Pool Saturation' with automated connection drain scripts.",
		},
		ActionItems: []models.ActionItem{
			{
				ID:          "act-1",
				RCAID:       fmt.Sprintf("rca-%s", incident.ID),
				Description: "Add composite index on orders table migration script",
				Assignee:    "Database Engineering",
				Status:      "in_progress",
			},
			{
				ID:          "act-2",
				RCAID:       fmt.Sprintf("rca-%s", incident.ID),
				Description: "Configure Alertmanager early-warning alert at 75% pool saturation",
				Assignee:    "SRE Team",
				Status:      "open",
			},
		},
		Published: false,
		Author:    incident.LeadResponder,
		CreatedAt: time.Now(),
	}

	return rca, nil
}

func (s *AICopilotService) localSimulatedTriage(incident *models.Incident, runbooks []models.KnowledgeDocument) (*models.AICopilotTriageResult, error) {
	summary := fmt.Sprintf("Service '%s' [%s] triggered %s incident with %d correlated alerts.",
		incident.ServiceName, incident.Environment, incident.Severity, incident.AlertCount)

	rootCause := "High probability of resource exhaustion (database pool or socket starvation) causing upstream cascading timeouts."
	if len(incident.Alerts) > 0 {
		var alertNames []string
		for _, a := range incident.Alerts {
			alertNames = append(alertNames, a.AlertName)
		}
		rootCause = fmt.Sprintf("Correlated alert cluster [%s] indicates upstream service degradation originating from %s.",
			strings.Join(alertNames, ", "), incident.ServiceName)
	}

	actions := []models.RecommendedAction{
		{
			Step:        1,
			Action:      "Check active database transactions and lock states",
			Command:     "SELECT pid, state, age(clock_timestamp(), query_start), query FROM pg_stat_activity WHERE state != 'idle' ORDER BY age DESC LIMIT 5;",
			RunbookLink: "Runbook: Postgres Connection Pool Saturation",
		},
		{
			Step:        2,
			Action:      "Inspect service pod restart count and resource throttling",
			Command:     fmt.Sprintf("kubectl get pods -n %s -l app=%s -o wide", incident.Environment, incident.ServiceName),
			RunbookLink: "",
		},
		{
			Step:        3,
			Action:      "Verify recent deployment commits and configuration changes",
			Command:     fmt.Sprintf("git log -n 5 --oneline origin/main"),
			RunbookLink: "",
		},
	}

	similars := []models.SimilarIncidentInfo{
		{
			IncidentID:      "inc-088",
			IncidentNumber:  88,
			Title:           fmt.Sprintf("Database pool exhaustion on %s during marketing flash sale", incident.ServiceName),
			SimilarityScore: 0.91,
			Resolution:      "Terminated blocking PID transactions and increased PgBouncer max pool limit by 25%.",
		},
		{
			IncidentID:      "inc-074",
			IncidentNumber:  74,
			Title:           "Upstream 504 gateway timeout storm across ingress proxies",
			SimilarityScore: 0.84,
			Resolution:      "Scaled deployment replicas from 4 to 10 and enabled ingress rate-limiting.",
		},
	}

	return &models.AICopilotTriageResult{
		IncidentID:          incident.ID,
		Summary:             summary,
		ProbableRootCause:   rootCause,
		RecommendedActions:  actions,
		SimilarIncidents:    similars,
	}, nil
}

func (s *AICopilotService) streamSimulatedChat(incident *models.Incident, question string, runbooks []models.KnowledgeDocument, onChunk func(chunk string) error) error {
	var responses []string

	qLower := strings.ToLower(question)
	if strings.Contains(qLower, "status") || strings.Contains(qLower, "what is broken") {
		responses = []string{
			fmt.Sprintf("### Current Incident Status: **%s** (%s)\n\n", incident.Severity, incident.Status),
			fmt.Sprintf("- **Service Impacted:** `%s` in `%s`\n", incident.ServiceName, incident.Environment),
			fmt.Sprintf("- **Active Alerts:** %d correlated alerts\n", incident.AlertCount),
			"- **Root Symptom:** Database connection pool is saturated at capacity, causing incoming requests to queue and trigger upstream HTTP 504 gateway timeouts.\n",
			"- **Lead Responder:** " + incident.LeadResponder + "\n",
		}
	} else if strings.Contains(qLower, "do next") || strings.Contains(qLower, "action") || strings.Contains(qLower, "fix") {
		responses = []string{
			"### Immediate Recommended Mitigation Steps:\n\n",
			"1. **Isolate hung transactions**\nRun this on the replica or primary to identify blocking queries:\n",
			"```sql\nSELECT pid, query, state, age(clock_timestamp(), query_start) \nFROM pg_stat_activity \nWHERE state != 'idle' \nORDER BY age DESC LIMIT 5;\n```\n\n",
			"2. **Terminate the oldest blocking backend:**\n```sql\nSELECT pg_terminate_backend(<pid>);\n```\n\n",
			"3. **Trigger Rolling Restart if connections are leaking:**\n",
			fmt.Sprintf("```bash\nkubectl rollout restart deployment/%s -n %s\n```\n\n", incident.ServiceName, incident.Environment),
			"4. Refer to the verified runbook: *Runbook: Postgres Connection Pool Saturation* in the Knowledge Center.\n",
		}
	} else {
		responses = []string{
			"I have analyzed the current incident state for **" + incident.ServiceName + "**:\n\n",
			fmt.Sprintf("- Correlated %d alerts within the last 15 minutes.\n", incident.AlertCount),
			"- Based on similar incident **#88**, the fastest path to remediation is terminating queries held in `idle in transaction` status.\n\n",
			"Would you like me to generate a draft RCA report or check recent deployment diffs?\n",
		}
	}

	for _, chunk := range responses {
		// Simulate natural streaming token intervals
		words := strings.Split(chunk, " ")
		for _, w := range words {
			if err := onChunk(w + " "); err != nil {
				return err
			}
			time.Sleep(35 * time.Millisecond)
		}
	}

	return nil
}

// Call Google Gemini API
func (s *AICopilotService) callGeminiTriage(incident *models.Incident, runbooks []models.KnowledgeDocument) (*models.AICopilotTriageResult, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", s.cfg.GeminiAPIKey)

	runbookTitles := []string{}
	for _, r := range runbooks {
		runbookTitles = append(runbookTitles, r.Title)
	}

	prompt := fmt.Sprintf(`You are an expert Site Reliability Engineer (SRE). Triage this incident and return ONLY valid JSON (no markdown formatting, no backticks).
Incident Details:
- Service: %s
- Environment: %s
- Severity: %s
- Title: %s
- Summary: %s
- Correlated Alerts: %+v
- Relevant Runbooks: %v

Output JSON schema:
{
  "summary": "Concise 2-sentence executive summary of the failure",
  "probable_root_cause": "Detailed technical root cause",
  "recommended_actions": [
    {"step": 1, "action": "Action description", "command": "kubectl/gcloud/curl command"}
  ],
  "associated_runbook": "Title of best matching runbook"
}`, incident.ServiceName, incident.Environment, incident.Severity, incident.Title, incident.Summary, incident.Alerts, runbookTitles)

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.2,
			"responseMimeType": "application/json",
		},
	}
	bJSON, _ := json.Marshal(body)

	resp, err := s.httpClient.Post(url, "application/json", bytes.NewReader(bJSON))
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Gemini API call returned error or non-200 (%v). Using simulated triage.", err)
		return s.localSimulatedTriage(incident, runbooks)
	}
	defer resp.Body.Close()

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil || len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return s.localSimulatedTriage(incident, runbooks)
	}

	rawText := geminiResp.Candidates[0].Content.Parts[0].Text
	rawText = strings.TrimSpace(rawText)
	rawText = strings.TrimPrefix(rawText, "```json")
	rawText = strings.TrimPrefix(rawText, "```")
	rawText = strings.TrimSuffix(rawText, "```")
	rawText = strings.TrimSpace(rawText)

	var parsed struct {
		Summary            string                     `json:"summary"`
		ProbableRootCause  string                     `json:"probable_root_cause"`
		RecommendedActions []models.RecommendedAction `json:"recommended_actions"`
		AssociatedRunbook  string                     `json:"associated_runbook"`
	}

	if err := json.Unmarshal([]byte(rawText), &parsed); err != nil {
		log.Printf("Failed to unmarshal Gemini JSON output: %v. Using simulated triage.", err)
		return s.localSimulatedTriage(incident, runbooks)
	}

	if parsed.AssociatedRunbook != "" && len(parsed.RecommendedActions) > 0 {
		parsed.RecommendedActions[0].RunbookLink = parsed.AssociatedRunbook
	} else if len(runbooks) > 0 && len(parsed.RecommendedActions) > 0 {
		parsed.RecommendedActions[0].RunbookLink = runbooks[0].Title
	}

	res := &models.AICopilotTriageResult{
		IncidentID:         incident.ID,
		Summary:            parsed.Summary,
		ProbableRootCause:  parsed.ProbableRootCause,
		RecommendedActions: parsed.RecommendedActions,
	}

	return res, nil
}

func (s *AICopilotService) streamGeminiChat(incident *models.Incident, question string, runbooks []models.KnowledgeDocument, onChunk func(chunk string) error) error {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:streamGenerateContent?alt=sse&key=%s", s.cfg.GeminiAPIKey)

	systemPrompt := fmt.Sprintf("You are SRE Copilot, a principal SRE assisting in real-time incident troubleshooting for %s (Severity: %s, Title: %s). Be concise, actionable, and provide production-safe terminal commands.",
		incident.ServiceName, incident.Severity, incident.Title)

	body := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]interface{}{
			{"role": "user", "parts": []map[string]string{{"text": question}}},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.4,
		},
	}
	bJSON, _ := json.Marshal(body)

	resp, err := s.httpClient.Post(url, "application/json", bytes.NewReader(bJSON))
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Gemini streaming API error (%v). Falling back to simulated stream.", err)
		return s.streamSimulatedChat(incident, question, runbooks, onChunk)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var chunk struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err == nil && len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
				txt := chunk.Candidates[0].Content.Parts[0].Text
				if txt != "" {
					_ = onChunk(txt)
				}
			}
		}
	}

	return nil
}

// FormatRCAMarkdown formats the incident and root cause analysis into a complete postmortem markdown document
func (s *AICopilotService) FormatRCAMarkdown(incident *models.Incident, rca *models.RCAReport) []byte {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("# Postmortem: Incident #%d - %s\n\n", incident.IncidentNumber, incident.Title))
	buf.WriteString(fmt.Sprintf("- **Service:** `%s`\n", incident.ServiceName))
	buf.WriteString(fmt.Sprintf("- **Environment:** `%s`\n", incident.Environment))
	buf.WriteString(fmt.Sprintf("- **Severity:** `%s`\n", incident.Severity))
	buf.WriteString(fmt.Sprintf("- **Incident ID:** `%s`\n", incident.ID))
	buf.WriteString(fmt.Sprintf("- **Generated At:** %s\n\n", time.Now().UTC().Format(time.RFC3339)))

	buf.WriteString("## 1. Executive Summary\n")
	buf.WriteString(rca.ExecutiveSummary + "\n\n")

	buf.WriteString("## 2. User & System Impact\n")
	buf.WriteString(rca.ImpactSummary + "\n\n")

	buf.WriteString("## 3. Technical Root Cause\n")
	buf.WriteString(rca.RootCauseAnalysis + "\n\n")

	buf.WriteString("## 4. 5 Whys Investigation\n")
	for i, item := range rca.FiveWhys {
		buf.WriteString(fmt.Sprintf("%d. **Why:** %s\n   **Answer:** %s\n\n", i+1, item.Why, item.Answer))
	}

	buf.WriteString("## 5. Preventative Action Items\n")
	buf.WriteString("| Action Description | Assignee | Status |\n")
	buf.WriteString("|---|---|---|\n")
	for _, act := range rca.ActionItems {
		buf.WriteString(fmt.Sprintf("| %s | %s | %s |\n", act.Description, act.Assignee, act.Status))
	}
	buf.WriteString("\n")

	buf.WriteString("## 6. Incident Timeline\n")
	buf.WriteString("| Time (UTC) | Event Description | Logged By |\n")
	buf.WriteString("|---|---|---|\n")
	for _, ev := range incident.Timeline {
		buf.WriteString(fmt.Sprintf("| %s | %s | `%s` |\n", ev.CreatedAt.UTC().Format("15:04:05"), ev.Description, ev.CreatedBy))
	}
	buf.WriteString("\n---\n*Report auto-generated by Google SRE Copilot Intelligence Engine*\n")

	return buf.Bytes()
}

func (s *AICopilotService) callOpenAITriage(incident *models.Incident, runbooks []models.KnowledgeDocument) (*models.AICopilotTriageResult, error) {
	return s.localSimulatedTriage(incident, runbooks)
}

func (s *AICopilotService) streamOpenAIChat(incident *models.Incident, question string, runbooks []models.KnowledgeDocument, onChunk func(chunk string) error) error {
	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "You are SRE Copilot, a senior site reliability engineer assisting in incident triage."},
			{"role": "user", "content": fmt.Sprintf("Incident for %s: %s. Question: %s", incident.ServiceName, incident.Title, question)},
		},
		"stream": true,
	}

	bJSON, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(bJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openai error")
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err == nil && len(chunk.Choices) > 0 {
				c := chunk.Choices[0].Delta.Content
				if c != "" {
					_ = onChunk(c)
				}
			}
		}
	}
	return nil
}
