# Helm provider.
provider "helm" {
  kubernetes {
    host                   = module.gke.endpoint
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(module.gke.ca_certificate)
  }
}

# GCR admin service account.
module "gcr_admin_service_account" {
  source     = "terraform-google-modules/service-accounts/google"
  version    = "4.0.0"
  depends_on = [google_project_service.services]

  project_id    = var.project.id
  names         = ["dexly-gcr-admin"]
  generate_keys = true
}

# Make sure the GCR backing bucket exists before assigning IAM roles.
resource "google_container_registry" "registry" {}

# Set GCR admin service account as storage admin of the GCR backend bucket.
resource "google_storage_bucket_iam_member" "gcr_admin" {
  bucket = google_container_registry.registry.id
  role   = "roles/storage.admin"
  member = module.gcr_admin_service_account.iam_email
}

# Google Kubernetes Engine cluster.
module "gke" {
  source     = "terraform-google-modules/kubernetes-engine/google//modules/private-cluster"
  version    = "14.1.0"
  depends_on = [module.vpc]

  project_id               = var.project.id
  name                     = var.gke_cluster.name
  region                   = var.region.name
  regional                 = false
  zones                    = var.region.zones
  network                  = var.vpc_network.name
  subnetwork               = var.gke_subnet.name
  ip_range_pods            = var.gke_subnet.pods_ip_range_name
  ip_range_services        = var.gke_subnet.services_ip_range_name
  remove_default_node_pool = true
  enable_private_nodes     = true

  node_pools = [
    {
      name         = var.gke_cluster.node_pool_name
      machine_type = var.gke_cluster.node_pool_machine_type
      autoscaling  = false
      node_count   = var.gke_cluster.node_pool_size
    }
  ]
}

# Install ArgoCD on the cluster using Helm.
resource "helm_release" "argo" {
  name       = "argo"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argo-cd"
  version    = "3.2.2"
}

# Deploy ArgoCD applications chart.
resource "helm_release" "argo_applications" {
  name  = "argo-applications"
  chart = "../helm/argo-applications"
}