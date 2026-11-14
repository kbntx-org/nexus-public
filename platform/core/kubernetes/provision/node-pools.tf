locals {
  node_pools = {
    "default" = {
      count       = 3
      server_type = "cx33"
      location    = "fsn1"
      labels      = {}
      taints      = []
    }
    "default-nbg1" = {
      count       = 1
      server_type = "cx33"
      location    = "nbg1"
      labels      = {}
      taints      = []
    }
    "workspace" = {
      count       = 1
      server_type = "cx33"
      location    = "hel1"
      labels = {
        "node-type"      = "workspace"
        "sysbox-install" = "yes"
      }
      taints = [
        {
          key    = "dedicated"
          value  = "workspace"
          effect = "NoSchedule"
        }
      ]
    }
  }

  # Taints travel as "taint-<key>" Hetzner labels so the Ansible k3s role can register them on the node.
  worker_nodes = merge([
    for pool_key, pool in local.node_pools : {
      for index in range(pool.count) : "${pool_key}-${index}" => {
        name        = "${local.cluster_name}-${pool_key}-${index}"
        pool_key    = pool_key
        server_type = pool.server_type
        location    = pool.location
        labels = merge(pool.labels, {
          for taint in pool.taints : "taint-${taint.key}" => "${taint.value}.${taint.effect}"
        })
      }
    }
  ]...)
}

resource "hcloud_server" "node_pool" {
  for_each = local.worker_nodes

  name                     = each.value.name
  server_type              = each.value.server_type
  image                    = "ubuntu-24.04"
  location                 = each.value.location
  keep_disk                = true
  shutdown_before_deletion = true
  user_data                = data.cloudinit_config.worker.rendered
  placement_group_id       = hcloud_placement_group.cluster_placement_group.id

  network {
    network_id = data.hcloud_network.main_network.id
    alias_ips  = []
  }

  public_net {
    ipv4_enabled = false
    ipv6_enabled = false
  }

  labels = merge({
    "type"         = "worker"
    "pool"         = each.value.pool_key
    "cluster-name" = local.cluster_name
  }, each.value.labels)

  lifecycle {
    ignore_changes = [user_data, ssh_keys]
  }
}

data "cloudinit_config" "worker" {
  gzip          = false
  base64_encode = false

  part {
    content_type = "text/cloud-config"
    filename     = "cloud-config.yml"
    content = templatefile("${path.module}/config/cloud-init.yml", merge(local.cloud_init_base, {
      type = "worker"
    }))
  }
}
