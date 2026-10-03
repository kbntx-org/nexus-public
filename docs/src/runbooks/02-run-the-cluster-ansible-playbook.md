---
title: Run the cluster Ansible playbook
---

Apply the k3s <a href="https://docs.ansible.com/" target="_blank" rel="noopener">Ansible</a> role to
the static nodes Terraform created: install k3s on a new node, or roll out a change to its
configuration. The playbook only manages static nodes (see
[Cluster & Compute](../platform/cluster/01-overview.md#cluster-shape)).

## When to run it

- A change to the
  <a href="https://github.com/kbntx-org/nexus/tree/main/platform/modules/k3s/ansible/roles/k3s" target="_blank" rel="noopener"><code>k3s
  role</code></a>, its templates, or
  <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/kubernetes/configuration/group_vars/all.yml" target="_blank" rel="noopener"><code>group_vars/all.yml</code></a>.
- A new static node, after Terraform created it.
- A k3s token change: read the
  [token gotcha](../platform/cluster/01-overview.md#gotcha-changing-the-agent-token-doesnt-touch-the-datastore)
  first, the playbook alone is not enough.

**Not for k3s upgrades.** The role installs k3s only when the binary is missing, so bumping
`k3s_version` leaves existing nodes alone; upgrades go through the system-upgrade-controller (see
[Provisioning and upgrades](../platform/cluster/01-overview.md#provisioning-and-upgrades)). Keep
`k3s_version` in line with the upgrade plans anyway, since new nodes install it.

## Before you start

- **WARP connected.** The inventory connects to each node's private IP as `engineer`, with the
  short-lived certificate Cloudflare Access issues (see
  [SSH to a node](../platform/traffic/01-overview.md#ssh-to-a-node)). Check with
  `ssh engineer@<node-private-ip>` before running anything.
- **Python and <a href="https://docs.astral.sh/uv/" target="_blank" rel="noopener">uv</a>** from the
  root `mise.toml` (`mise install`). Entering the configuration folder with mise activated creates
  and activates its `.venv`.
- **`vars.yaml`** in the configuration folder (gitignored), holding the secrets the inventory and
  the role need:

  ```yaml
  hcloud_token: <hetzner-api-token>
  k3s_token: <server-token>
  k3s_agent_token: <agent-token>
  # Optional, raises Docker Hub pull limits:
  k3s_docker_hub_username: <username>
  k3s_docker_hub_password: <access-token>
  ```

## Steps

Every command runs from `platform/core/kubernetes/configuration/`; the
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/kubernetes/configuration/pyproject.toml" target="_blank" rel="noopener"><code>pyproject.toml</code></a>
tasks append extra arguments to the `ansible-playbook` call.

1. Install Ansible and the Galaxy collections:

   ```sh
   uv sync
   uv run poe install
   ```

2. Check the inventory lists the nodes you expect, in the right group:

   ```sh
   uv run poe inventory
   ```

3. Dry-run on one node and read the diff:

   ```sh
   uv run poe check --limit <node-name>
   ```

4. Apply to that node, then to everything:

   ```sh
   uv run poe playbook --limit <node-name>
   uv run poe playbook
   ```

   Start with a worker: the control plane restarting takes the API down for a moment.

## Verify

```sh
ssh engineer@<control-plane-private-ip> sudo k3s kubectl get nodes -o wide
ssh engineer@<node-private-ip> systemctl status k3s        # on a server
ssh engineer@<node-private-ip> systemctl status k3s-agent  # on a worker
```

Every node is `Ready`, with the labels and taints it should have.

## Gotchas

- **One node at a time.** The playbook runs servers first, then workers, with `serial: 1`. A node
  restarts k3s only when its config changed, but with a single server that still means a short API
  outage; workloads keep running.
- **Karpenter nodes are excluded** from the inventory by their label and never see the role. Their
  k3s config is a copy in the
  <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/karpenter/templates/secrets.yaml" target="_blank" rel="noopener">Karpenter
  cloud-init</a>: change both together, and replace running Karpenter nodes to pick it up.
- **`--diff` prints secrets.** The rendered k3s config holds the tokens; don't paste the output
  anywhere.
- **Check mode is meaningless on a fresh node:** it skips the install, then fails on the missing
  service. Use it on nodes that already run k3s.
- **The server group must have an odd size** for etcd quorum; the role asserts it before changing
  anything.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/kubernetes/configuration" target="_blank" rel="noopener"><code>platform/core/kubernetes/configuration/</code></a>
  — playbook, dynamic Hetzner inventory, `ansible.cfg` and Python project
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/kubernetes/configuration/inventory/hcloud.yml" target="_blank" rel="noopener"><code>platform/core/kubernetes/configuration/inventory/hcloud.yml</code></a>
  — how nodes are grouped, and how Karpenter nodes are left out
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/modules/k3s/ansible/roles/k3s" target="_blank" rel="noopener"><code>platform/modules/k3s/ansible/roles/k3s/</code></a>
  — the role: tasks, defaults, config templates
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/kubernetes/upgrades" target="_blank" rel="noopener"><code>platform/core/kubernetes/upgrades/</code></a>
  — the k3s upgrade plans
