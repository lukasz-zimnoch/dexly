# Terraform config.
terraform {
  backend "gcs" {
    bucket = "dexly-terraform-backend-bucket"
    prefix = "terraform/state"
  }

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "3.62.0"
    }

    helm = {
      source = "hashicorp/helm"
      version = "2.1.2"
    }
  }
}

# Google client config data.
data "google_client_config" "default" {}

# Google provider.
provider "google" {
  project = var.project.id
  region  = var.region.name
  zone    = var.region.zones[0]
}

# Project services and APIs.
resource "google_project_service" "services" {
  for_each                   = toset(var.services)
  service                    = each.value
  disable_dependent_services = true
}