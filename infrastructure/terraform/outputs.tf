output "gcr_admin_key" {
  description = "GCR admin account key"
  value       = module.gcr_admin_service_account.key
}