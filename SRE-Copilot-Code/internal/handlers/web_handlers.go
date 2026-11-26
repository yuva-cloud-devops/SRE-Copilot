package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"sre-copilot/internal/models"
	"sre-copilot/internal/services"
)

type WebHandler struct {
	incidentSvc  *services.IncidentService
	knowledgeSvc *services.KnowledgeService
	aiSvc        *services.AICopilotService
}

func NewWebHandler(incidentSvc *services.IncidentService, knowledgeSvc *services.KnowledgeService, aiSvc *services.AICopilotService) *WebHandler {
	return &WebHandler{
		incidentSvc:  incidentSvc,
		knowledgeSvc: knowledgeSvc,
		aiSvc:        aiSvc,
	}
}

// DashboardPage renders the main operational view
func (h *WebHandler) DashboardPage(c *gin.Context) {
	incidents, err := h.incidentSvc.List("", "")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load incidents: %v", err)
		return
	}

	p1Count, p2Count, p3Count, resolvedCount := 0, 0, 0, 0
	for _, inc := range incidents {
		if inc.Status == models.StatusResolved {
			resolvedCount++
		} else {
			switch inc.Severity {
			case models.SeverityP1:
				p1Count++
			case models.SeverityP2:
				p2Count++
			case models.SeverityP3, models.SeverityP4:
				p3Count++
			}
		}
	}

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"Title":         "Operational Dashboard",
		"ActiveTab":     "dashboard",
		"Incidents":     incidents,
		"P1Count":       p1Count,
		"P2Count":       p2Count,
		"P3Count":       p3Count,
		"ResolvedCount": resolvedCount,
		"TotalCount":    len(incidents),
	})
}

// IncidentsPage renders the dedicated incident management list with filters
func (h *WebHandler) IncidentsPage(c *gin.Context) {
	statusFilter := c.Query("status")
	sevFilter := c.Query("severity")

	incidents, err := h.incidentSvc.List(models.IncidentStatus(statusFilter), models.IncidentSeverity(sevFilter))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load incidents: %v", err)
		return
	}

	c.HTML(http.StatusOK, "incidents.html", gin.H{
		"Title":          "Incident Management",
		"ActiveTab":      "incidents",
		"Incidents":      incidents,
		"StatusFilter":   statusFilter,
		"SeverityFilter": sevFilter,
		"TotalCount":     len(incidents),
	})
}

// IncidentDetailPage renders the split-screen investigation workspace
func (h *WebHandler) IncidentDetailPage(c *gin.Context) {
	id := c.Param("id")
	incident, err := h.incidentSvc.Get(id)
	if err != nil {
		c.String(http.StatusNotFound, "Incident not found")
		return
	}

	// Retrieve runbooks matching service
	runbooks, _ := h.knowledgeSvc.Search("", incident.ServiceName, 3)

	c.HTML(http.StatusOK, "incident_detail.html", gin.H{
		"Title":     incident.Title,
		"ActiveTab": "incidents",
		"Incident":  incident,
		"Runbooks":  runbooks,
	})
}

// RCAListPage renders the postmortems & RCA center index
func (h *WebHandler) RCAListPage(c *gin.Context) {
	incidents, err := h.incidentSvc.List("", "")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load incidents: %v", err)
		return
	}

	var rcaList []map[string]interface{}
	for _, inc := range incidents {
		rca, _ := h.aiSvc.GenerateRCA(&inc)
		rcaList = append(rcaList, map[string]interface{}{
			"Incident":  inc,
			"RCA":       rca,
			"ActionCnt": len(rca.ActionItems),
		})
	}

	c.HTML(http.StatusOK, "rca_list.html", gin.H{
		"Title":     "RCA Center & Postmortems",
		"ActiveTab": "rca",
		"RCAList":   rcaList,
	})
}

// RCAPage renders the postmortem generator and editor for a specific incident
func (h *WebHandler) RCAPage(c *gin.Context) {
	id := c.Param("id")
	incident, err := h.incidentSvc.Get(id)
	if err != nil {
		c.String(http.StatusNotFound, "Incident not found")
		return
	}

	rca, err := h.aiSvc.GenerateRCA(incident)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate RCA: %v", err)
		return
	}

	c.HTML(http.StatusOK, "rca_editor.html", gin.H{
		"Title":     "RCA: " + incident.Title,
		"ActiveTab": "rca",
		"Incident":  incident,
		"RCA":       rca,
	})
}

// KnowledgePage renders the runbook & postmortem search explorer
func (h *WebHandler) KnowledgePage(c *gin.Context) {
	query := c.Query("q")
	service := c.Query("service")

	docs, err := h.knowledgeSvc.Search(query, service, 10)
	if err != nil {
		docs = []models.KnowledgeDocument{}
	}

	c.HTML(http.StatusOK, "knowledge.html", gin.H{
		"Title":     "Knowledge Base & Runbooks",
		"ActiveTab": "knowledge",
		"Query":     query,
		"Service":   service,
		"Documents": docs,
	})
}

// DownloadRCAMarkdown downloads the formatted postmortem markdown file
func (h *WebHandler) DownloadRCAMarkdown(c *gin.Context) {
	id := c.Param("id")
	incident, err := h.incidentSvc.Get(id)
	if err != nil {
		c.String(http.StatusNotFound, "Incident not found")
		return
	}

	rca, err := h.aiSvc.GenerateRCA(incident)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate RCA: %v", err)
		return
	}

	mdBytes := h.aiSvc.FormatRCAMarkdown(incident, rca)
	filename := fmt.Sprintf("incident-%d-postmortem.md", incident.IncidentNumber)

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", mdBytes)
}
