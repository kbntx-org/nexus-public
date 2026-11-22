---
title: Kubernetes
---

The cluster is k3s on Hetzner. Everything below needs WARP and is for platform work; apps reach the
cluster only through [ArgoCD](../argocd/how-to.md).

## Use kubectl

[SSH to the control plane](../cloudflare/how-to.md#ssh-to-a-node) as `engineer`: the k3s role
installs a kubeconfig for that user. To use it from your machine over WARP, copy it and set `server`
to the control plane's private IP (it points at `127.0.0.1`). ArgoCD reverts any change you make by
hand unless the app's auto-sync is paused.

## Upgrade k3s

- **Patch:** Renovate opens the PR; review and merge it. Check that the role's `group_vars/all.yml`,
  which new nodes install, has the same version as the upgrade plans.
- **Minor:** first, in its own PR, add the new minor to `is_supported_k8s_version()` in
  <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/sysbox/scripts/sysbox-deploy-k8s.sh" target="_blank" rel="noopener"><code>sysbox-deploy-k8s.sh</code></a>
  and merge it, or the Sysbox installer exits on the workspace node. Its image keeps the same tag,
  so roll it out by hand once CI pushed it:
  `kubectl -n kube-system rollout restart ds/sysbox-deploy-k8s`, and check the pod's logs on the
  workspace node. Then, in a second PR, bump the plans and `group_vars/all.yml`.

## Run the Ansible playbook

Run it after a change to the k3s role, its templates or `group_vars/all.yml`, or to set up a node
OpenTofu just created. Not for upgrades.

Before you start: WARP on (the inventory connects to private IPs as `engineer` with Cloudflare's
short-lived certificate; check with `ssh engineer@<node-private-ip>`), `mise install` for Python and
uv, and a gitignored `vars.yaml` in the configuration folder:

```yaml
hcloud_token: <hetzner-api-token>
k3s_token: <server-token>
k3s_agent_token: <agent-token>
k3s_zot_password: <node-password> # optional, lets nodes pull from zot
```

From `platform/core/kubernetes/configuration/`:

```sh
uv sync && uv run poe install         # Ansible and Galaxy collections
uv run poe inventory                  # check the nodes and their groups
uv run poe check --limit <node-name>  # dry run with a diff
uv run poe playbook --limit <node-name>
uv run poe playbook
```

Start with a worker. Nodes run one at a time, and a server restarts k3s only when its config
changed, which with one server is a short API outage. `--diff` prints the tokens; check mode fails
on a node without k3s. Verify with `sudo k3s kubectl get nodes -o wide` on the control plane: every
node `Ready`, with its labels and taints.

## Rotate the k3s token

Run `k3s token rotate -t <old> --new-token <new>` on a server, restart the servers and agents with
the new value, then update `vars.yaml`. Rotating to the value the config already has resyncs the
datastore without changing the token.

How it works: [Kubernetes design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/kubernetes/configuration/pyproject.toml" target="_blank" rel="noopener"><code>configuration/pyproject.toml</code></a>
  — the `poe` tasks
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/kubernetes/configuration/group_vars/all.yml" target="_blank" rel="noopener"><code>configuration/group_vars/all.yml</code></a>
  — k3s version and settings for new nodes
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/kubernetes/upgrades/plans/values.yaml" target="_blank" rel="noopener"><code>upgrades/plans/values.yaml</code></a>
  — the version the cluster runs
