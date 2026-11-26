package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"sre-copilot/internal/models"
	"sre-copilot/internal/services"
)

type APIHandler struct {
	incidentSvc  *services.IncidentService
	knowledgeSvc *services.KnowledgeService
	aiSvc        *services.AICopilotService
	storageSvc   *services.StorageService
	broadcaster  *services.SSEBroadcaster
}

func NewAPIHandler(incidentSvc *services.IncidentService, knowledgeSvc *services.KnowledgeService, aiSvc *services.AICopilotService, storageSvc *services.StorageService, broadcaster *services.SSEBroadcaster) *APIHandler {
	return &APIHandler{
		incidentSvc:  incidentSvc,
		knowledgeSvc: knowledgeSvc,
		aiSvc:        aiSvc,
		storageSvc:   storageSvc,
		broadcaster:  broadcaster,
	}
}

// SSEGlobalStream handles real-time timeline & incident push to connected clients
func (h *APIHandler) SSEGlobalStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	ch := h.broadcaster.Subscribe()
	defer h.broadcaster.Unsubscribe(ch)

	c.Stream(func(w io.Writer) bool {
		msg, ok := <-ch
		if !ok {
			return false
		}
		_, _ = fmt.Fprint(w, msg)
		return true
	})
}

// AITriage generates incident analysis
func (h *APIHandler) AITriage(c *gin.Context) {
	id := c.Param("id")
	incident, err := h.incidentSvc.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found"})
		return
	}

	runbooks, _ := h.knowledgeSvc.Search("", incident.ServiceName, 3)
	result, err := h.aiSvc.TriageIncident(incident, runbooks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// AIChatStream streams conversational SRE advice token by token via SSE
func (h *APIHandler) AIChatStream(c *gin.Context) {
	id := c.Param("id")
	incident, err := h.incidentSvc.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found"})
		return
	}

	var req struct {
		Question string `json:"question"`
	}
	if err := c.BindJSON(&req); err != nil || req.Question == "" {
		req.Question = "What should I do next?"
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	runbooks, _ := h.knowledgeSvc.Search("", incident.ServiceName, 3)

	err = h.aiSvc.StreamChat(incident, req.Question, runbooks, func(chunk string) error {
		payload, _ := json.Marshal(map[string]string{"chunk": chunk})
		c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", payload)))
		c.Writer.Flush()
		return nil
	})

	if err != nil {
		c.Writer.Write([]byte("data: {\"error\": \"stream failed\"}\n\n"))
		c.Writer.Flush()
	}
}

// UpdateIncidentStatus handles HTMX status updates
func (h *APIHandler) UpdateIncidentStatus(c *gin.Context) {
	id := c.Param("id")
	status := c.PostForm("status")
	responder := c.PostForm("responder")
	if responder == "" {
		responder = "Alex (On-Call SRE)"
	}

	inc, err := h.incidentSvc.UpdateStatus(id, models.IncidentStatus(status), responder)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, inc)
}

// AddComment handles adding manual notes
func (h *APIHandler) AddComment(c *gin.Context) {
	id := c.Param("id")
	comment := c.PostForm("comment")
	author := c.PostForm("author")

	if comment == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment cannot be empty"})
		return
	}

	event, err := h.incidentSvc.AddComment(id, comment, author)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, event)
}

// PublishRCA handles publishing the Postmortem / RCA report to Google Cloud Storage
func (h *APIHandler) PublishRCA(c *gin.Context) {
	id := c.Param("id")
	incident, err := h.incidentSvc.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found"})
		return
	}

	rca, err := h.aiSvc.GenerateRCA(incident)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate RCA: %v", err)})
		return
	}

	mdBytes := h.aiSvc.FormatRCAMarkdown(incident, rca)
	objectName := fmt.Sprintf("postmortems/incident-%d-rca.md", incident.IncidentNumber)

	uri, err := h.storageSvc.UploadArtifact(c.Request.Context(), objectName, "text/markdown; charset=utf-8", mdBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload to Cloud Storage: %v", err)})
		return
	}

	// Record publication event in the incident timeline
	_, _ = h.incidentSvc.AddComment(id, fmt.Sprintf("Published official Postmortem RCA to Cloud Storage: %s", uri), "SRE Postmortem Engine")

	c.JSON(http.StatusOK, gin.H{
		"status":      "published",
		"artifact":    uri,
		"object_name": objectName,
		"message":     fmt.Sprintf("Postmortem published to %s", uri),
	})
}
