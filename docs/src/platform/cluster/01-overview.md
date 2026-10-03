---
title: Overview
---

Nexus runs on a <a href="https://k3s.io/" target="_blank" rel="noopener">k3s</a> cluster on
<a href="https://www.hetzner.com/cloud" target="_blank" rel="noopener">Hetzner Cloud</a> VMs.

## Why k3s on Hetzner

A managed control plane is a great default, but this platform is also meant to teach how Kubernetes
works, not just how to use it. k3s is a low-footprint distribution, and Hetzner offers competitive
compute pricing in Europe.

The bundled k3s services (Traefik, servicelb, local-path storage, flannel, kube-proxy, network
policy controller, cloud controller) are disabled on purpose, so the cluster looks like vanilla
Kubernetes: ingress, CNI and storage come back as their own GitOps-managed components.

## Cluster shape

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/cluster-shape.svg"
</div>

What the diagram doesn't show is why:

- **Embedded etcd on a single server** lets the control plane grow to HA later without migrating the
  datastore.
- **Static pools** are cheap enough to run continuously, so they are sized once rather than scaled.
- **<a href="https://karpenter.sh/" target="_blank" rel="noopener">Karpenter</a> nodes** configure
  themselves at boot through their cloud-init, which is why they are excluded from the Ansible
  inventory. The controller comes with its
  <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/karpenter" target="_blank" rel="noopener">Hetzner
  provider</a>, and the cloud-init secret is rendered by
  [ESO templating](../secrets/01-overview.md#external-secrets-operator).

## Provisioning and upgrades

1. Terraform creates the static VMs
   (<a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/kubernetes/provision" target="_blank" rel="noopener"><code>provision/</code></a>)
   and an Ansible k3s
   <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/kubernetes/configuration" target="_blank" rel="noopener"><code>role</code></a>
   configures them; see
   [Run the cluster Ansible playbook](../../runbooks/02-run-the-cluster-ansible-playbook.md).
2. <a href="https://github.com/kbntx-org/nexus/blob/main/platform/modules/k3s/terraform/config/init-core-cluster-dependencies.sh" target="_blank" rel="noopener"><code>init-core-cluster-dependencies.sh</code></a>
   installs the critical components below.
3. Later k3s upgrades go through Rancher's
   <a href="https://github.com/rancher/system-upgrade-controller" target="_blank" rel="noopener">system-upgrade-controller</a>.

### Gotcha: changing the agent token doesn't touch the datastore

**Symptom:** after re-running the role with a new `k3s_agent_token`, the control plane fails to
restart, yet new nodes still join with the old token.

**Cause:** `k3s_token` and `k3s_agent_token` (consumed by the
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/modules/k3s/ansible/roles/k3s/templates/k3s-config.yml.j2" target="_blank" rel="noopener"><code>k3s
role's config template</code></a>) are written into the encrypted bootstrap data on the datastore
only when a server first initializes (`cluster-init`). Re-running the role updates
`/etc/rancher/k3s/config.yaml` on disk, but the datastore keeps the init-time value (the server
token itself, if no distinct agent token was passed yet). The on-disk config and the datastore then
disagree.

**Fix:** run <a href="https://docs.k3s.io/cli/token" target="_blank" rel="noopener"><code>k3s token
rotate</code></a> (`-t <old> --new-token <new>`) and restart the servers and agents with the new
value; this forces k3s to rewrite the bootstrap data. Rotating to the value the config already has
forces the same resync without changing the token.

## Core components

In the order they show up on fresh VMs:

- **<a href="https://helm.sh/docs/" target="_blank" rel="noopener">Helm</a>** installs the initial
  components below, then is only used for templating.

- **<a href="https://cilium.io/" target="_blank" rel="noopener">Cilium</a>** goes first, before even
  the cloud controller, because nothing schedules without a CNI. k3s runs with
  `disable-kube-proxy: true` and `disable-network-policy: true` so Cilium's eBPF dataplane replaces
  kube-proxy (better performance) and enforces `NetworkPolicy`/`CiliumNetworkPolicy` natively,
  instead of running a CNI plus a separate proxy layer.

- **<a href="https://github.com/hetznercloud/hcloud-cloud-controller-manager" target="_blank" rel="noopener">Hetzner
  Cloud Controller Manager</a>** integrates nodes with Hetzner (metadata, cleanup when a server is
  deleted), and
  **<a href="https://github.com/hetznercloud/csi-driver" target="_blank" rel="noopener">Hetzner
  CSI</a>** exposes Hetzner block storage as a storage class.

- **<a href="https://argo-cd.readthedocs.io/" target="_blank" rel="noopener">ArgoCD</a>** then takes
  over reconciling itself and everything else in the cluster; see
  [Delivery](../delivery/01-overview.md).

- **<a href="https://github.com/nestybox/sysbox" target="_blank" rel="noopener">Sysbox</a>** lets a
  pod run its own Docker daemon (Docker-in-Docker) inside a user namespace instead of needing
  `privileged: true`, which would hand it root on the node. ArgoCD deploys it as an installer
  DaemonSet on every node labeled `sysbox-install=yes`, registering a `sysbox-runc` `RuntimeClass`
  that pods opt into with `runtimeClassName`. The installer image is built in the repo because
  upstream only publishes it for amd64 and its script needed patching for k3s.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/kubernetes/provision" target="_blank" rel="noopener"><code>platform/core/kubernetes/provision/</code></a>
  — cluster-shape Terraform (control plane, static pools, firewalls)
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/modules/k3s/terraform" target="_blank" rel="noopener"><code>platform/modules/k3s/terraform/</code></a>
  — reusable VM-provisioning module + first-boot bootstrap script
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/modules/k3s/ansible" target="_blank" rel="noopener"><code>platform/modules/k3s/ansible/</code></a>
  — the `k3s` role that actually installs/configures k3s
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/kubernetes/configuration" target="_blank" rel="noopener"><code>platform/core/kubernetes/configuration/</code></a>
  — dynamic Hetzner inventory + playbook running the role
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/kubernetes/upgrades" target="_blank" rel="noopener"><code>platform/core/kubernetes/upgrades/</code></a>
  — system-upgrade-controller and the k3s upgrade plans
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/cilium" target="_blank" rel="noopener"><code>platform/core/cilium/</code></a>
  — CNI, kube-proxy replacement, NetworkPolicy engine
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/hetzner-cloud-controller" target="_blank" rel="noopener"><code>platform/core/hetzner-cloud-controller/</code></a>
  — Hetzner CCM + CSI Helm chart
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/karpenter" target="_blank" rel="noopener"><code>platform/core/karpenter/</code></a>
  — on-demand Hetzner node provisioning and the cloud-init `userData` secret
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/sysbox" target="_blank" rel="noopener"><code>platform/core/sysbox/</code></a>
  — Sysbox installer image and DaemonSet chart
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/network" target="_blank" rel="noopener"><code>platform/core/network/</code></a>
  — the private VPC the cluster joins
