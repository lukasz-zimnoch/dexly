provider "helm" {
  kubernetes {
    host                   = module.gke.endpoint
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(module.gke.ca_certificate)
  }
}

# ------------------------------------------------------------------------------
# Create a Google Kubernetes Engine private cluster.
# ------------------------------------------------------------------------------

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

# ------------------------------------------------------------------------------
# Setup the Google Container Registry. Create a GCR admin service account and
# make it an admin of the underlying bucket. Grant view access for the service
# account used by GKE.
# ------------------------------------------------------------------------------

module "gcr_admin_service_account" {
  source     = "terraform-google-modules/service-accounts/google"
  version    = "4.0.0"
  depends_on = [google_project_service.services]

  project_id    = var.project.id
  names         = ["dexly-gcr-admin"]
  generate_keys = true
}

resource "google_container_registry" "this" {}

resource "google_storage_bucket_iam_member" "gcr_admin_sa-gcr_admin" {
  bucket = google_container_registry.this.id
  role   = "roles/storage.admin"
  member = module.gcr_admin_service_account.iam_email
}

resource "google_storage_bucket_iam_member" "gke_sa-gcr_object_viewer" {
  bucket = google_container_registry.this.id
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${module.gke.service_account}"
}

# ------------------------------------------------------------------------------
# Deploy ArgoCD and setup it to manage project applications (services).
# ------------------------------------------------------------------------------

resource "helm_release" "argo_cd" {
  name       = "argocd"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argo-cd"
  version    = "3.2.2"
}

resource "helm_release" "argo_applications" {
  depends_on = [helm_release.argo_cd]
  name       = "argo-applications"
  chart      = "../helm/argo-applications"
}

# ------------------------------------------------------------------------------
# Deploy Postgres operator.
# ------------------------------------------------------------------------------

resource "helm_release" "postgres_operator" {
  name  = "postgres-operator"
  chart = "https://github.com/zalando/postgres-operator/raw/v1.6.2/charts/postgres-operator/postgres-operator-1.6.2.tgz"
}

# ------------------------------------------------------------------------------
# Workload identity configuration.
# ------------------------------------------------------------------------------

module "trading_workload_identity" {
  source     = "terraform-google-modules/kubernetes-engine/google//modules/workload-identity"
  version    = "15.0.0"
  depends_on = [module.gke]

  project_id = var.project.id
  name       = "trading"
  namespace  = "default"
}
