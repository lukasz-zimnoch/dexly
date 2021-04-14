module "vpc" {
  source     = "terraform-google-modules/network/google"
  version    = "3.2.0"
  depends_on = [google_project_service.services]

  project_id   = var.project.id
  network_name = var.vpc_network.name

  subnets = [
    {
      subnet_name   = var.gke_subnet.name
      subnet_ip     = var.gke_subnet.primary_ip_range
      subnet_region = var.region.name
    }
  ]

  secondary_ranges = {
    (var.gke_subnet.name) = [
      {
        range_name    = var.gke_subnet.pods_ip_range_name
        ip_cidr_range = var.gke_subnet.pods_ip_range
      },
      {
        range_name    = var.gke_subnet.services_ip_range_name
        ip_cidr_range = var.gke_subnet.services_ip_range
      }
    ]
  }
}
