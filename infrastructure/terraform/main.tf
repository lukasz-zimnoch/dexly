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
  }
}

# Providers.
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

# Make sure the GCR backing bucket exists before assigning IAM roles.
resource "google_container_registry" "registry" {}

# GCR admin service account.
module "gcr_admin_service_account" {
  source        = "terraform-google-modules/service-accounts/google"
  version       = "4.0.0"
  depends_on    = [google_project_service.services]

  project_id    = var.project.id
  names         = ["dexly-gcr-admin"]
  generate_keys = true
}

# Set GCR admin service account as storage admin of the GCR backend bucket.
resource "google_storage_bucket_iam_member" "gcr_admin" {
  bucket = google_container_registry.registry.id
  role = "roles/storage.admin"
  member = module.gcr_admin_service_account.iam_email
}