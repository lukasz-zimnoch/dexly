# ------------------------------------------------------------------------------
# Create a bucket for Cloud Functions archives.
# ------------------------------------------------------------------------------

resource "google_storage_bucket" "cloud_functions_archives" {
  name = var.cloud_functions.archives_bucket_name

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }
}

# ------------------------------------------------------------------------------
# Deploy notification function.
# ------------------------------------------------------------------------------

data "archive_file" "notification_function" {
  type        = "zip"
  source_dir  = "../../notification"
  output_path = "archive/${var.cloud_functions.notification_function_name}.zip"
}

resource "google_storage_bucket_object" "notification_function" {
  name   = "${var.cloud_functions.notification_function_name}.zip"
  bucket = google_storage_bucket.cloud_functions_archives.name
  source = data.archive_file.notification_function.output_path
}

resource "google_cloudfunctions_function" "notification" {
  name        = var.cloud_functions.notification_function_name
  runtime     = "go113"

  entry_point           = "ProcessEvent"
  available_memory_mb   = 128
  source_archive_bucket = google_storage_bucket.cloud_functions_archives.name
  source_archive_object = google_storage_bucket_object.notification_function.name

  event_trigger {
    event_type = "google.pubsub.topic.publish"
    resource   = google_pubsub_topic.notifications.name
  }

  environment_variables = {
    MAIL_PASSWORD = var.mail_config.password
  }
}