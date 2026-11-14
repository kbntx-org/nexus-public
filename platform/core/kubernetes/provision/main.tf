data "hcloud_network" "main_network" {
  name = "main-vpc"
}

data "hcloud_server" "bastion" {
  with_selector = "app=bastion"
}

locals {
  cluster_name = "nexus"

  cloud_init_base = {
    cloudflare_access_ssh_ca_public_key = file("${path.module}/config/ca.pub")
    init_script_b64                     = base64encode(file("${path.module}/config/init-core-cluster-dependencies.sh"))
    nat_ip                              = one(data.hcloud_server.bastion.network).ip
    # Hetzner gateway in vpc is on X.X.X.1
    network_gateway_ip = cidrhost(data.hcloud_network.main_network.ip_range, 1)
  }
}

resource "hcloud_placement_group" "cluster_placement_group" {
  name = "${local.cluster_name}-k3s-placement-group"
  type = "spread"
}

module "control_plane_firewall" {
  source = "../../../modules/firewall"
  name   = "nexus-control-plane-firewall"

  ingress_rules = [
    { protocol = "tcp", port = "22", source_ips = ["10.0.0.0/16"] },
    { protocol = "tcp", port = "6443", source_ips = ["10.0.0.0/16"] }
  ]

  label_selector = "type=control-plane,cluster-name=nexus"
}

data "cloudflare_zone" "kbntx" {
  filter = {
    name = "kbntx.com"
  }
}

resource "cloudflare_dns_record" "nexus_control_plane" {
  zone_id = data.cloudflare_zone.kbntx.zone_id
  name    = "nexus-control-plane"
  type    = "A"
  content = hcloud_server_network.control_plane_network.ip
  ttl     = 300
  proxied = false
}

module "worker_nodes_firewall" {
  source = "../../../modules/firewall"
  name   = "nexus-worker-nodes-firewall"

  ingress_rules = [
    { protocol = "tcp", port = "22", source_ips = ["10.0.0.0/16"] }
  ]

  label_selector = "type=worker,cluster-name=nexus"
}
