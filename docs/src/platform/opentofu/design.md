---
title: OpenTofu design
---

<a href="https://opentofu.org/docs/" target="_blank" rel="noopener">OpenTofu</a> manages everything
outside the cluster: the Hetzner network, servers and bastion, Cloudflare WARP and Access, and the
Vault auth setup. <a href="https://www.runatlantis.io/" target="_blank" rel="noopener">Atlantis</a>
plans and applies it from pull request comments.

## How it's set up

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/atlantis.svg"
</div>

Each stack is a folder in its component (`platform/core/network/`,
`platform/core/kubernetes/provision/`), registered as a project in `atlantis.yaml`, with its state
in R2. Atlantis runs in the cluster, from a hand-written chart, and only GitHub's webhook reaches it
from outside. Its credentials and stack variables (`TF_VAR_<variable>`) come from Vault.

## Decisions and gotchas

- **Atlantis over GitHub Actions jobs**: it locks a stack per PR and applies exactly the plan that
  was reviewed.
- **Autoplan is off and there are no apply requirements**: plans run on demand, and with a single
  maintainer no review could satisfy a requirement.
- **Atlantis runs on the cluster it can change.** A `kubernetes` plan that replaces the node
  Atlantis runs on would kill the apply: apply such a plan from your machine. A `user_data` change
  never replaces a node (`ignore_changes`).
- **`required_version` is pinned to the `opentofu` version in `mise.toml`**: Atlantis installs the
  version `required_version` names, your shell the one in `mise.toml`, and Renovate bumps both in
  one PR.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/atlantis.yaml" target="_blank" rel="noopener"><code>atlantis.yaml</code></a>
  — the projects Atlantis plans
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/services/atlantis" target="_blank" rel="noopener"><code>platform/services/atlantis/</code></a>
  — chart, server-side repo config, routes and secrets
