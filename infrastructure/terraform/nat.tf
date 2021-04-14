module "cloud_nat" {
  source     = "terraform-google-modules/cloud-nat/google"
  version    = "2.0.0"
  depends_on = [module.vpc]

  project_id = var.project.id
  name       = var.cloud_nat.name
  region     = var.region.name
  network    = var.vpc_network.name

  create_router = true
  router        = var.cloud_nat.router_name

  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"

  subnetworks = [
    {
      name                     = var.gke_subnet.name
      source_ip_ranges_to_nat  = ["LIST_OF_SECONDARY_IP_RANGES"]
      secondary_ip_range_names = [var.gke_subnet.pods_ip_range_name]
    }
  ]
}