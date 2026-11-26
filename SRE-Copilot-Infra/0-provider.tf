terraform {
  required_version = ">= 1.5.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.20"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 5.20"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

# Decode the central YAML configuration file
locals {
  cfg = yamldecode(file("${path.module}/${var.config_file}"))
}

provider "google" {
  project = local.cfg.project.id
  region  = local.cfg.project.region
  zone    = local.cfg.project.zone
}

provider "google-beta" {
  project = local.cfg.project.id
  region  = local.cfg.project.region
  zone    = local.cfg.project.zone
}

resource "random_id" "suffix" {
  byte_length = 4
}
