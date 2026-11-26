package services

import (
	"sre-copilot/internal/models"
	"sre-copilot/internal/repository"
)

type KnowledgeService struct {
	repo repository.KnowledgeRepository
}

func NewKnowledgeService(repo repository.KnowledgeRepository) *KnowledgeService {
	return &KnowledgeService{repo: repo}
}

func (k *KnowledgeService) Search(query string, serviceName string, limit int) ([]models.KnowledgeDocument, error) {
	if limit <= 0 {
		limit = 5
	}
	return k.repo.Search(query, serviceName, limit)
}

func (k *KnowledgeService) ListAll() ([]models.KnowledgeDocument, error) {
	return k.repo.ListAll()
}

func (k *KnowledgeService) Get(id string) (*models.KnowledgeDocument, error) {
	return k.repo.GetDocument(id)
}

func (k *KnowledgeService) Add(doc *models.KnowledgeDocument) error {
	return k.repo.AddDocument(doc)
}
