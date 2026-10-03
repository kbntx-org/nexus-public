# Nexus

My personal internal developer platform to deploy apps in production and experiment with modern
tooling and technologies.

The platform is GitOps-driven: Terraform provisions a k3s cluster on Hetzner, and ArgoCD reconciles
everything running on it from this repo. It hosts the apps published under
[kbntx.com](https://kbntx.com).

## What's inside

- [apps/](apps/) — the product applications the platform hosts.
- [platform/](platform/) — everything that makes the platform run: `core/` for cluster-level
  building blocks, `services/` for workloads that aren't product apps.
- [docs/](docs/) — the documentation site, published at [docs.kbntx.com](https://docs.kbntx.com).

## Stack

- **Apps**: Angular, React, Go, TypeScript, Nx monorepo
- **Infrastructure**: Hetzner Cloud, k3s, Cilium, Karpenter, Terraform, Ansible
- **GitOps & CI/CD**: ArgoCD (app-of-apps), GitHub Actions
- **Traffic & access**: Cloudflare Tunnel and Zero Trust, Traefik, cert-manager, external-dns
- **Secrets**: Vault, External Secrets Operator
- **Databases**: CloudNativePG
- **Observability**: Grafana, VictoriaMetrics, Loki

## Quick start

The only prerequisite is Docker; every other tool is pinned in [mise.toml](mise.toml).

```sh
platform/core/local/local.sh create   # install the tools, create the local kind cluster
mise run dev docs                     # start an app on top of core infra → http://docs.localhost
```

[Local development](https://docs.kbntx.com/getting-started/02-local-development/) covers how it fits
together, tear-down and gotchas.

## Documentation

Design decisions, how each part of the platform fits together, gotchas and runbooks live at
[docs.kbntx.com](https://docs.kbntx.com).

## License

[MIT](LICENSE).
