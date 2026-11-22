---
title: Overview
---

Nexus is a personal internal developer platform: a place to run apps in production and try modern
tooling. It is a <a href="https://k3s.io/" target="_blank" rel="noopener">k3s</a> cluster on
<a href="https://www.hetzner.com/cloud" target="_blank" rel="noopener">Hetzner Cloud</a>,
provisioned with <a href="https://opentofu.org/docs/" target="_blank" rel="noopener">OpenTofu</a>
and kept in the state this repository declares by
<a href="https://argo-cd.readthedocs.io/" target="_blank" rel="noopener">ArgoCD</a>.

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/platform-overview.svg"
</div>

Two rules shape the rest: the cluster has no open inbound ports, and secrets never live in Git.

## Where to start

- **Run it on your machine:** [Local environment](../platform/local-environment/how-to.md).
- **Understand a part of the platform:** every tool under Platform has a **Design** page (what it
  is, why we chose it, how it fits) and a **How to** page (tasks only, platform-only tasks last).

## Ship an app

Copy the closest existing app,
<a href="https://github.com/kbntx-org/nexus/tree/main/platform/services/kiln" target="_blank" rel="noopener"><code>platform/services/kiln</code></a>
for one with an image, and go through these in order:

1. [Make it build an image](../platform/ci/how-to.md#make-a-project-build-and-deploy).
2. Write its chart, one template file per resource kind, with a
   [network policy](../platform/cilium/how-to.md#write-a-network-policy).
3. [Give it secrets](../platform/vault/how-to.md#use-a-secret-in-a-chart) and, if needed,
   [a database](../platform/postgres/how-to.md#add-a-database-to-an-app).
4. [Route traffic to it](../platform/traefik/how-to.md#route-traffic-to-an-app).
5. [Run it locally](../platform/local-environment/how-to.md#add-an-app-to-tilt).
6. [Deploy it with ArgoCD](../platform/argocd/how-to.md#add-an-app).
7. Optionally, [add a dashboard](../platform/grafana/how-to.md#add-a-dashboard).

Ongoing work is tracked as
<a href="https://github.com/kbntx-org/nexus/issues" target="_blank" rel="noopener">GitHub
issues</a>; the docs describe the platform as it is today.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core" target="_blank" rel="noopener"><code>platform/core/</code></a>
  — cluster-level building blocks: networking, certificates, secrets, ingress, operators
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/services" target="_blank" rel="noopener"><code>platform/services/</code></a>
  — workloads hosted on the platform that are not product apps
- <a href="https://github.com/kbntx-org/nexus/blob/main/mise.toml" target="_blank" rel="noopener"><code>mise.toml</code></a>
  — pinned version of every tool
