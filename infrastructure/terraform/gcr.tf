resource "google_container_registry" "registry" {}

module "gcr_admin_service_account" {
  source        = "terraform-google-modules/service-accounts/google"
  version       = "4.0.0"

  project_id    = var.project.id
  names         = ["gcr-admin"]
}

resource "google_storage_bucket_iam_member" "gcr_admin" {
  bucket = google_container_registry.registry.id
  role = "roles/storage.admin"
  member = module.gcr_admin_service_account.iam_email
}