package config

import (
	"encoding/json"
	"log"
	"os"
)

type ConfigFile struct {
	Server struct {
		Port        string `json:"port"`
		Environment string `json:"environment"`
		AppName     string `json:"app_name"`
	} `json:"server"`
	Database struct {
		DatabaseURL            string `json:"database_url"`
		MaxOpenConns           int    `json:"max_open_conns"`
		MaxIdleConns           int    `json:"max_idle_conns"`
		ConnMaxLifetimeMinutes int    `json:"conn_max_lifetime_minutes"`
	} `json:"database"`
	Storage struct {
		BucketName string `json:"bucket_name"`
	} `json:"storage"`
	GCP struct {
		ProjectID           string `json:"project_id"`
		DefaultRegion       string `json:"default_region"`
		AlertingWebhookPath string `json:"alerting_webhook_path"`
		LoggingWebhookPath  string `json:"logging_webhook_path"`
	} `json:"gcp"`
	AI struct {
		Provider             string `json:"provider"`
		GeminiAPIKey         string `json:"gemini_api_key"`
		OpenAIAPIKey         string `json:"openai_api_key"`
		ModelName            string `json:"model_name"`
		SimulateTokenDelayMS int    `json:"simulate_token_delay_ms"`
	} `json:"ai"`
	Correlation struct {
		SlidingWindowMinutes     int  `json:"sliding_window_minutes"`
		AutoGroupByService       bool `json:"auto_group_by_service"`
		DeduplicateByFingerprint bool `json:"deduplicate_by_fingerprint"`
	} `json:"correlation"`
}

type Config struct {
	Port                     string
	Environment              string
	AppName                  string
	DatabaseURL              string
	DBMaxOpenConns           int
	DBMaxIdleConns           int
	DBConnMaxLifetimeMinutes int
	StorageBucketName        string
	GCPProjectID             string
	GCPRegion                string
	AIProvider               string // "gemini", "openai", "mock"
	GeminiAPIKey             string
	OpenAIAPIKey             string
	ModelName                string
	SlidingWindowMinutes     int
	SimulateTokenDelayMS     int
	VectorDim                int
}

func LoadConfig() *Config {
	cfg := &Config{
		Port:                     "8080",
		Environment:              "development",
		AppName:                  "SRE Copilot",
		DatabaseURL:              "postgres://postgres:postgres@localhost:5432/sre_copilot?sslmode=disable",
		DBMaxOpenConns:           25,
		DBMaxIdleConns:           5,
		DBConnMaxLifetimeMinutes: 30,
		GCPProjectID:             "prod-ecommerce-project",
		GCPRegion:                "us-central1",
		AIProvider:               "mock",
		ModelName:                "gemini-1.5-flash",
		SlidingWindowMinutes:     15,
		SimulateTokenDelayMS:     30,
		VectorDim:                1536,
	}

	// 1. Try reading from config.json
	fileData, err := os.ReadFile("config.json")
	if err == nil {
		var jsonCfg ConfigFile
		if err := json.Unmarshal(fileData, &jsonCfg); err == nil {
			log.Println("✓ Loaded configuration from config.json")
			if jsonCfg.Server.Port != "" {
				cfg.Port = jsonCfg.Server.Port
			}
			if jsonCfg.Server.Environment != "" {
				cfg.Environment = jsonCfg.Server.Environment
			}
			if jsonCfg.Server.AppName != "" {
				cfg.AppName = jsonCfg.Server.AppName
			}
			if jsonCfg.Database.DatabaseURL != "" {
				cfg.DatabaseURL = jsonCfg.Database.DatabaseURL
			}
			if jsonCfg.Database.MaxOpenConns > 0 {
				cfg.DBMaxOpenConns = jsonCfg.Database.MaxOpenConns
			}
			if jsonCfg.Database.MaxIdleConns > 0 {
				cfg.DBMaxIdleConns = jsonCfg.Database.MaxIdleConns
			}
			if jsonCfg.Database.ConnMaxLifetimeMinutes > 0 {
				cfg.DBConnMaxLifetimeMinutes = jsonCfg.Database.ConnMaxLifetimeMinutes
			}
			if jsonCfg.Storage.BucketName != "" {
				cfg.StorageBucketName = jsonCfg.Storage.BucketName
			}
			if jsonCfg.GCP.ProjectID != "" {
				cfg.GCPProjectID = jsonCfg.GCP.ProjectID
			}
			if jsonCfg.GCP.DefaultRegion != "" {
				cfg.GCPRegion = jsonCfg.GCP.DefaultRegion
			}
			if jsonCfg.AI.Provider != "" {
				cfg.AIProvider = jsonCfg.AI.Provider
			}
			if jsonCfg.AI.GeminiAPIKey != "" {
				cfg.GeminiAPIKey = jsonCfg.AI.GeminiAPIKey
			}
			if jsonCfg.AI.OpenAIAPIKey != "" {
				cfg.OpenAIAPIKey = jsonCfg.AI.OpenAIAPIKey
			}
			if jsonCfg.AI.ModelName != "" {
				cfg.ModelName = jsonCfg.AI.ModelName
			}
			if jsonCfg.AI.SimulateTokenDelayMS > 0 {
				cfg.SimulateTokenDelayMS = jsonCfg.AI.SimulateTokenDelayMS
			}
			if jsonCfg.Correlation.SlidingWindowMinutes > 0 {
				cfg.SlidingWindowMinutes = jsonCfg.Correlation.SlidingWindowMinutes
			}
		} else {
			log.Printf("Warning: Failed to parse config.json: %v", err)
		}
	} else {
		log.Println("Notice: config.json not found, reading environment variables and defaults")
	}

	// 2. Allow OS Environment Variables to override if present
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}
	if env := os.Getenv("APP_ENV"); env != "" {
		cfg.Environment = env
	}
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}
	if bucket := os.Getenv("STORAGE_BUCKET_NAME"); bucket != "" {
		cfg.StorageBucketName = bucket
	}
	if gcpProject := os.Getenv("GCP_PROJECT_ID"); gcpProject != "" {
		cfg.GCPProjectID = gcpProject
	}
	if provider := os.Getenv("AI_PROVIDER"); provider != "" {
		cfg.AIProvider = provider
	}
	if geminiKey := os.Getenv("GEMINI_API_KEY"); geminiKey != "" {
		cfg.GeminiAPIKey = geminiKey
		if cfg.AIProvider == "mock" {
			cfg.AIProvider = "gemini"
		}
	}
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		cfg.OpenAIAPIKey = openaiKey
		if cfg.AIProvider == "mock" && cfg.GeminiAPIKey == "" {
			cfg.AIProvider = "openai"
		}
	}

	return cfg
}
