---
title: Kubernetes design
---

The cluster is <a href="https://k3s.io/" target="_blank" rel="noopener">k3s</a> on
<a href="https://www.hetzner.com/cloud" target="_blank" rel="noopener">Hetzner Cloud</a> VMs in a
private network. A managed control plane would be simpler, but this platform is also meant to teach
how Kubernetes works; k3s is light and Hetzner is cheap in Europe.

## How it's set up

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/cluster-shape.svg"
</div>

1. **OpenTofu** creates the control plane and the static node pools in `provision/`.
2. **An Ansible k3s role** installs and configures k3s on them, see
   [Run the Ansible playbook](how-to.md#run-the-ansible-playbook).
3. **`init-core-cluster-dependencies.sh`** installs, with Helm, what nothing can schedule without:
   [Cilium](../cilium/design.md) first, then the Hetzner cloud controller and CSI driver (node
   metadata, block storage), then ArgoCD, which takes over everything, itself included.
4. **The system-upgrade-controller** rolls out later k3s versions from the upgrade plans.

## Decisions and gotchas

- **k3s's bundled add-ons are disabled** (Traefik, servicelb, local-path, flannel, kube-proxy,
  network policy controller, cloud controller), so the cluster looks like vanilla Kubernetes and
  each piece comes back as its own GitOps-managed chart.
- **Embedded etcd on one server**, so the control plane can grow to HA without migrating the
  datastore. The server group must stay odd for quorum; the role checks it.
- **Static pools, no autoscaler**: Hetzner bills by the hour, so on-demand nodes cost more than a
  pool sized once.
- **Taints travel as Hetzner labels** (`taint-<key>`), which the role registers on the node, as for
  the dedicated [workspace](../workspace/design.md) pool.
- **`k3s_token` and `k3s_agent_token` live in the datastore**, written once when the server first
  initializes. Changing them in the role only updates the config on disk: the control plane then
  fails to restart while nodes still join with the old token. See
  [Rotate the k3s token](how-to.md#rotate-the-k3s-token).
- **Kubernetes Secrets are encrypted in etcd** (`secrets-encryption: true`).
- **The role installs k3s only when it is missing**, so bumping `k3s_version` there changes only new
  nodes. Keep it equal to the upgrade plans.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/kubernetes/provision" target="_blank" rel="noopener"><code>platform/core/kubernetes/provision/</code></a>
  — control plane, node pools, firewall and first-boot script
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/kubernetes/configuration" target="_blank" rel="noopener"><code>platform/core/kubernetes/configuration/</code></a>
  — inventory, playbook and the `k3s` role
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/kubernetes/upgrades" target="_blank" rel="noopener"><code>platform/core/kubernetes/upgrades/</code></a>
  — system-upgrade-controller and upgrade plans
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/hetzner-cloud-controller" target="_blank" rel="noopener"><code>platform/core/hetzner-cloud-controller/</code></a>
  — Hetzner cloud controller and CSI driver
