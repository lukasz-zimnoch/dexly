resource "google_pubsub_topic" "notifications" {
  name = var.pubsub.notifications_topic_name
}