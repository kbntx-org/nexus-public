---
title: ArgoCD design
---

<a href="https://argo-cd.readthedocs.io/" target="_blank" rel="noopener">ArgoCD</a> keeps the
cluster in the state `main` declares. Every workload exists because ArgoCD applied it from this
repository: a manual `kubectl apply` is reverted on the next sync, Git history is the deploy history
and a rollback is a `git revert`.

## How it's set up

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/delivery-paths.svg"
</div>

A **config change** (a Helm value, a manifest) is synced from `main` with no build. A **source
change** to something this repository builds an image for goes through [CI](../ci/design.md) first,
which then bumps the image tag (see [Deploys](../deploys/design.md)).

One root `Application`, created by hand once, renders the `app-of-apps` chart: the two projects, an
admission policy and one `ApplicationSet` that generates an `Application` per environment of every
`argo.yaml` in the repository. Bootstrapping the platform is one sync of the root. `prod` keeps the
bare app name and namespace; any other environment adds `-<env>` to both.

| Project    | Holds                       | Permissions                                                               |
| ---------- | --------------------------- | ------------------------------------------------------------------------- |
| `platform` | Every chart outside `apps/` | Any namespace and resource, cluster-scoped included                       |
| `product`  | Charts under `apps/`        | Its listed namespaces, whitelisted kinds, no `ResourceQuota`/`LimitRange` |

Apps that ship an image use a multi-source `Application`: the chart from this repository, plus the
values file named by `manifestsValues` from
<a href="https://github.com/kbntx-org/nexus-manifests" target="_blank" rel="noopener"><code>nexus-manifests</code></a>,
where CI writes the tag. ArgoCD polls both repositories, so a change lands within a few minutes.

## Decisions and gotchas

- **One `argo.yaml` per chart instead of a central list**, so a chart's deployment lives next to it
  and new environments derive from it.
- **A `ValidatingAdmissionPolicy` ties `product` to `apps/`**, and forbids moving an existing
  `Application` into `product`, because an `AppProject` restricts repositories, not paths: a wrong
  project would silently give a product chart platform permissions. CEL runs in the API server, so
  there is no webhook server or certificate to operate.
- **A kind outside the `product` whitelist fails to sync and is hidden from the resource tree**,
  even when an operator created it. That's why the whole core group is open: the Pods behind a
  `Deployment` would not show otherwise.
- **Auto-sync can be switched off in the UI**, and it sticks, as an incident switch. Git no longer
  shows it, so each `Application` carries a `nexus/auto-sync` label from its `argo.yaml`, and the
  **Argo CD - Applications** dashboard lists apps whose live setting differs.
- **Deleting an `Application` deletes what it deployed** (the foreground resources finalizer),
  unless `deleteResources: false`. Removing an `argo.yaml`, an environment, or renaming `app` does
  exactly that: a stateful app loses its data. Deleting the `ApplicationSet` itself deletes nothing
  (`preserveResourcesOnDeletion`).
- **Login is GitHub SSO through Dex**, with a GitHub team mapped to `role:admin`; the local `admin`
  account is disabled.
- **The reconcile and git duration histograms top out at 16s and 20s**: a slower call shows at the
  ceiling, not its real duration.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/services/app-of-apps" target="_blank" rel="noopener"><code>platform/services/app-of-apps/</code></a>
  — projects, admission policy and the `ApplicationSet`
- <a href="https://github.com/kbntx-org/nexus/blob/main/.yaml-schemas/argo.schema.json" target="_blank" rel="noopener"><code>.yaml-schemas/argo.schema.json</code></a>
  — every key an `argo.yaml` can set
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/argocd" target="_blank" rel="noopener"><code>platform/core/argocd/</code></a>
  — ArgoCD chart, SSO, RBAC and dashboard
