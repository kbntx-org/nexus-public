---
title: Overview
---

## What is Nexus?

Nexus is a personal internal developer platform to deploy apps in production and experiment with
modern tooling and technologies. It runs on a
<a href="https://k3s.io/" target="_blank" rel="noopener">k3s</a> cluster on
<a href="https://www.hetzner.com/cloud" target="_blank" rel="noopener">Hetzner Cloud</a>,
provisioned with
<a href="https://developer.hashicorp.com/terraform" target="_blank" rel="noopener">Terraform</a> and
reconciled continuously from Git via
<a href="https://argo-cd.readthedocs.io/" target="_blank" rel="noopener">ArgoCD</a>.

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/platform-overview.svg"
</div>

The cluster has no open inbound ports, and secrets stay out of Git. Each domain page below explains
how, and why each component was chosen.

## Where to go next

- [Local development](02-local-development.md) — prerequisites and how to run the whole stack on
  your machine.
- [Cluster & Compute](../platform/cluster/01-overview.md) — provisioning, upgrades, CNI,
  autoscaling.
- [Traffic & Access](../platform/traffic/01-overview.md) — how a request or an operator reaches a
  cluster with no open inbound ports.
- [Delivery](../platform/delivery/01-overview.md) — GitOps, CI/CD, how a commit turns into a running
  change.
- [Databases](../platform/databases/01-overview.md) — the shared Postgres pattern and its backups.
- [Secrets](../platform/secrets/01-overview.md) — Vault + External Secrets, keeping values out of
  Git.
- [Observability](../platform/observability/01-overview.md) — metrics, logs, dashboards.
- [Runbooks](../runbooks/01-restore-a-cnpg-backup.md) — step-by-step procedures for operating the
  platform.

Ongoing work and planned improvements are tracked as
<a href="https://github.com/kbntx-org/nexus/issues" target="_blank" rel="noopener">GitHub issues</a>
— the docs describe the platform as it stands today, not the roadmap.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/app-of-apps/values.yaml" target="_blank" rel="noopener"><code>platform/services/app-of-apps/values.yaml</code></a>
  — every ArgoCD application deployed to the cluster
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core" target="_blank" rel="noopener"><code>platform/core/</code></a>
  — cluster-level building blocks: networking, certificates, secrets, ingress, operators
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/kubernetes" target="_blank" rel="noopener"><code>platform/core/kubernetes/</code></a>
  — Terraform provisioning and configuration of the k3s cluster
- <a href="https://github.com/kbntx-org/nexus/blob/main/mise.toml" target="_blank" rel="noopener"><code>mise.toml</code></a>
  — pinned versions of every runtime and tool
