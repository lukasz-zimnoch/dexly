# Terraform config.
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "3.62.0"
    }
  }
}

variable "project" {
  description = "Project ID"
  default     = "dexly-309412"
}

# Provider.
provider "google" {
  project = var.project
}

# Terraform backend storage bucket.
resource "google_storage_bucket" "terraform_backend" {
  name     = "dexly-terraform-backend-bucket"

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }
}