---
title: Workspace design
---

The workspace is an always-on development pod in the cluster: Claude Code, code-server and a
<a href="https://github.com/pingdotgg/t3code" target="_blank" rel="noopener">T3 Code</a> server on
one persistent home folder, so work continues from any device, the T3 Code mobile app included.

## How it's set up

The pod runs under [Sysbox](../sysbox/design.md), with its own Docker daemon started by the image's
entrypoint, on a dedicated `workspace` node tainted so nothing else lands there: its memory use
would otherwise squeeze production workloads. Everything is reachable only over WARP: code-server
and T3 Code on their own hosts, and a key-only SSH server, on a non-default port with keys from
Vault, through the Service's cluster DNS name.

## Decisions and gotchas

- **One volume for the whole home folder**, shared with Docker's data folder: the checkout, the
  Claude login, tool installs, images and containers survive a pod restart. Without it Sysbox keeps
  Docker's data on the node's disk and deletes it with the pod.
- **No auto-sync** (`autoSync: false` in its `argo.yaml`): its live state changes by hand, so a
  deploy reaches `nexus-manifests` but rolls out only on a manual sync from the ArgoCD UI.
- **Tools come from `mise.toml`**, like everywhere else: the bootstrap installs them and links
  Docker Compose and Buildx into Docker's CLI plugins folder.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/services/workspace" target="_blank" rel="noopener"><code>platform/services/workspace/</code></a>
  — image, chart and bootstrap script
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/kubernetes/provision/node-pools.tf" target="_blank" rel="noopener"><code>node-pools.tf</code></a>
  — the tainted `workspace` pool
