resource "google_pubsub_topic" "notifications" {
  name = var.pubsub.notifications_topic_name
}

resource "google_pubsub_topic_iam_member" "trading_sa-notifications_publisher" {
  topic  = google_pubsub_topic.notifications.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${module.trading_workload_identity.gcp_service_account_email}"
}
