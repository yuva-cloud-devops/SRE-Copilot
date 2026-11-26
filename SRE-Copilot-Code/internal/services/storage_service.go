package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"sre-copilot/internal/config"
)

type StorageService struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewStorageService(cfg *config.Config) *StorageService {
	return &StorageService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadArtifact uploads data to the provisioned Google Cloud Storage bucket.
// It uses instance metadata OAuth2 credentials if running in GCP, or falls back to local storage if running off-cloud.
func (s *StorageService) UploadArtifact(ctx context.Context, objectName string, contentType string, data []byte) (string, error) {
	bucket := s.cfg.StorageBucketName
	if bucket == "" {
		bucket = "sre-copilot-artifacts"
	}

	// 1. Try obtaining access token from GCP Compute Engine Metadata Server
	token, err := s.getMetadataToken(ctx)
	if err != nil {
		log.Printf("GCP metadata token unavailable (%v). Saving artifact locally to data/artifacts/%s", err, objectName)
		return s.saveLocalArtifact(objectName, data)
	}

	// 2. Upload directly to Google Cloud Storage JSON API
	uploadURL := fmt.Sprintf("https://storage.googleapis.com/upload/storage/v1/b/%s/o?uploadType=media&name=%s",
		url.PathEscape(bucket),
		url.QueryEscape(objectName),
	)

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create GCS upload request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload artifact to GCS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GCS API error (%d): %s", resp.StatusCode, string(respBody))
	}

	gcsURI := fmt.Sprintf("gs://%s/%s", bucket, objectName)
	log.Printf("✓ Successfully uploaded postmortem artifact to GCS: %s", gcsURI)
	return gcsURI, nil
}

// getMetadataToken fetches an OAuth2 access token for the default VM Service Account
func (s *StorageService) getMetadataToken(ctx context.Context) (string, error) {
	metadataURL := "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token"
	req, err := http.NewRequestWithContext(ctx, "GET", metadataURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata server returned status %d", resp.StatusCode)
	}

	var res struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.AccessToken, nil
}

// saveLocalArtifact stores artifacts locally as fallback when outside GCP
func (s *StorageService) saveLocalArtifact(objectName string, data []byte) (string, error) {
	localDir := filepath.Join("data", "artifacts", filepath.Dir(objectName))
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create local artifact directory: %w", err)
	}

	filePath := filepath.Join("data", "artifacts", objectName)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write local artifact: %w", err)
	}

	return "file://" + filePath, nil
}
