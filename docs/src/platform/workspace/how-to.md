---
title: Workspace
---

The workspace is a development pod in the cluster with Claude Code, code-server and T3 Code. Connect
WARP before anything below.

## Connect

- **Browser:** open code-server on its host (`ingressHost` in the chart values).
- **SSH:** connect to the Service's cluster DNS name on the SSH port from
  <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/workspace/templates/service.yaml" target="_blank" rel="noopener"><code>service.yaml</code></a>
  with a key from Vault. Start a `tmux` session to keep a terminal across disconnects.
- **T3 Code app:** run `t3 pair` in the workspace and enter the one-time code it prints on the
  device.

## Deploy a new workspace image

Merge the change; CI builds the image and bumps its tag in `nexus-manifests`. Then sync the
`workspace` app from the ArgoCD UI: it doesn't auto-sync. The pod restarts, and the home folder
stays.

How it works: [Workspace design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/workspace/files/bootstrap.sh" target="_blank" rel="noopener"><code>files/bootstrap.sh</code></a>
  — what starts in the pod
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/workspace/values.yaml" target="_blank" rel="noopener"><code>values.yaml</code></a>
  — hosts and image
