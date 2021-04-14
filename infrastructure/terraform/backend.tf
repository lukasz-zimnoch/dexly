terraform {
  backend "gcs" {
    bucket = "dexly-tf-backend-bucket"
    prefix = "terraform/state"
  }
}