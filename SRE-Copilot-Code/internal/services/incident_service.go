package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"sre-copilot/internal/models"
	"sre-copilot/internal/repository"
)

type IncidentService struct {
	repo        repository.IncidentRepository
	broadcaster *SSEBroadcaster
}

func NewIncidentService(repo repository.IncidentRepository, broadcaster *SSEBroadcaster) *IncidentService {
	return &IncidentService{
		repo:        repo,
		broadcaster: broadcaster,
	}
}

func (s *IncidentService) List(status models.IncidentStatus, severity models.IncidentSeverity) ([]models.Incident, error) {
	return s.repo.ListIncidents(status, severity)
}

func (s *IncidentService) Get(id string) (*models.Incident, error) {
	return s.repo.GetIncident(id)
}

func (s *IncidentService) UpdateStatus(id string, newStatus models.IncidentStatus, responder string) (*models.Incident, error) {
	inc, err := s.repo.GetIncident(id)
	if err != nil {
		return nil, err
	}

	oldStatus := inc.Status
	inc.Status = newStatus
	now := time.Now()

	if newStatus == models.StatusResolved {
		inc.ResolvedAt = &now
	}

	if responder != "" {
		inc.LeadResponder = responder
	}

	if err := s.repo.UpdateIncident(inc); err != nil {
		return nil, err
	}

	eventDesc := fmt.Sprintf("Incident status changed from %s to %s by %s", oldStatus, newStatus, responder)
	_ = s.repo.AddTimelineEvent(&models.TimelineEvent{
		ID:          uuid.New().String(),
		IncidentID:  id,
		EventType:   models.EventStatusChanged,
		Description: eventDesc,
		CreatedBy:   responder,
		CreatedAt:   now,
	})

	s.broadcaster.Broadcast("status_change", fmt.Sprintf(`{"incident_id":"%s","status":"%s"}`, id, newStatus))

	return inc, nil
}

func (s *IncidentService) AddComment(id string, comment string, author string) (*models.TimelineEvent, error) {
	if author == "" {
		author = "On-Call Responder"
	}

	event := &models.TimelineEvent{
		ID:          uuid.New().String(),
		IncidentID:  id,
		EventType:   models.EventCommentAdded,
		Description: comment,
		CreatedBy:   author,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.AddTimelineEvent(event); err != nil {
		return nil, err
	}

	s.broadcaster.Broadcast("timeline_update", fmt.Sprintf(`{"incident_id":"%s","comment":"%s"}`, id, comment))
	return event, nil
}
