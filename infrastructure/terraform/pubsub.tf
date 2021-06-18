resource "google_pubsub_topic" "notifications" {
  name = var.pubsub.notifications_topic_name
}

resource "google_pubsub_topic_iam_member" "gke_sa-notifications_publisher" {
  topic  = google_pubsub_topic.notifications.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${module.gke.service_account}"
}
