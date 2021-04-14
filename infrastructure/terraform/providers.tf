terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "3.62.0"
    }
  }
}

provider "google" {
  project = var.project.id
  region  = var.region.name
  zone    = var.region.zones[0]
}

