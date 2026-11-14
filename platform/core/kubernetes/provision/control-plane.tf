resource "random_password" "cluster_token" {
  length  = 64
  special = false
  upper   = true
  lower   = true
  numeric = true
}

resource "hcloud_server" "control_plane" {
  name                     = "${local.cluster_name}-k3s-control-plane"
  server_type              = "cx33"
  image                    = "ubuntu-24.04"
  location                 = "fsn1"
  keep_disk                = true
  shutdown_before_deletion = true
  user_data                = data.cloudinit_config.control_plane.rendered
  placement_group_id       = hcloud_placement_group.cluster_placement_group.id

  network {
    network_id = data.hcloud_network.main_network.id
    alias_ips  = []
  }

  public_net {
    ipv4_enabled = false
    ipv6_enabled = false
  }

  labels = {
    "type"         = "control-plane"
    "cluster-name" = local.cluster_name
    "node-type"    = "control-plane"
    "environment"  = "production"
  }

  lifecycle {
    ignore_changes = [user_data, ssh_keys]
  }
}

resource "hcloud_server_network" "control_plane_network" {
  server_id  = hcloud_server.control_plane.id
  network_id = data.hcloud_network.main_network.id
}

data "cloudinit_config" "control_plane" {
  gzip          = false
  base64_encode = false

  part {
    content_type = "text/cloud-config"
    filename     = "cloud-config.yml"
    content = templatefile("${path.module}/config/cloud-init.yml", merge(local.cloud_init_base, {
      type = "control-plane"
    }))
  }
}

output "cluster_token" {
  value     = random_password.cluster_token.result
  sensitive = true
}
