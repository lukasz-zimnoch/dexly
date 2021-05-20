variable "services" {
  type = list(string)
  default = [
    "iam.googleapis.com",
    "compute.googleapis.com",
    "container.googleapis.com",
  ]
}

variable "project" {
  type        = map(string)
  description = "Project info"

  default = {
    name = "dexly"
    id   = "dexly-309412"
  }
}

variable "region" {
  type        = object({name = string, zones = list(string)})
  description = "Region and zones info"

  default = {
    name  = "europe-central2"
    zones = ["europe-central2-a", "europe-central2-b", "europe-central2-c"]
  }
}

variable "vpc_network" {
  type        = map(string)
  description = "VPC network data"

  default = {
    name = "dexly-vpc-network"
  }
}

variable "gke_subnet" {
  type        = map(string)
  description = "Subnet for deploying GKE cluster resources"

  default = {
    name             = "dexly-gke-subnet"
    primary_ip_range = "10.1.0.0/16"

    pods_ip_range_name = "dexly-gke-pods-ip-range"
    pods_ip_range      = "10.100.0.0/16"

    services_ip_range_name = "dexly-gke-services-ip-range"
    services_ip_range      = "10.101.0.0/16"
  }
}

variable "gke_cluster" {
  type        = map(string)
  description = "GKE cluster info"

  default = {
    name                   = "dexly-gke-cluster"
    node_pool_name         = "dexly-gke-node-pool"
    node_pool_machine_type = "e2-small"
    node_pool_size         = 1
  }
}

variable "cloud_nat" {
  type        = map(string)
  description = "Cloud NAT info"

  default = {
    name        = "dexly-cloud-nat"
    router_name = "dexly-cloud-router"
  }
}