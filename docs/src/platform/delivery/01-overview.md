---
title: Overview
---

<a href="https://argo-cd.readthedocs.io/" target="_blank" rel="noopener">ArgoCD</a> is the GitOps
reconciler that owns the cluster. Every workload that runs in Nexus exists because ArgoCD read its
definition from this repo and applied it: the cluster state is a function of `main`. A manual
`kubectl apply` is reverted on the next reconcile, rollback is one `git revert`, and Git history
_is_ the deploy history.

## Two paths in, depending on what changed

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/delivery-paths.svg"
</div>

A **config change** (a Helm value, a new manifest) to a component that ships no image needs no build
or deploy: ArgoCD syncs it from `main`, and CI only runs its checks. This covers most of the
platform, including every third-party component wrapped in an umbrella chart (ArgoCD itself, for
example).

A **source change**, to anything this repo builds an image for, goes through CI first: build the
image, then hand ArgoCD a new tag. See [CI/CD pipeline](02-ci-cd-pipeline.md) and
[GitOps deploys](03-gitops-deploys.md).

## The app-of-apps pattern

A single root `Application`, registered by hand once at bootstrap, points at the
<a href="https://github.com/kbntx-org/nexus/tree/main/platform/services/app-of-apps" target="_blank" rel="noopener"><code>app-of-apps</code></a>
chart, a thin wrapper around
<a href="https://github.com/argoproj/argo-helm/tree/main/charts/argocd-apps" target="_blank" rel="noopener"><code>argocd-apps</code></a>.
It declares the two projects, the admission policy below and one
<a href="https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/" target="_blank" rel="noopener"><code>ApplicationSet</code></a>,
`applications`, that generates every other `Application` from the `argo.yaml` files of the repo.
Bootstrapping the platform is one sync of the root.

### argo.yaml

Each deployed Helm chart has an `argo.yaml` next to its `Chart.yaml`, validated in the editor by
<a href="https://github.com/kbntx-org/nexus/blob/main/.yaml-schemas/argo.schema.json" target="_blank" rel="noopener"><code>argo.schema.json</code></a>:

```yaml
app: smelt # Application name, unique across the repo
project: product # product under apps/, platform anywhere else
environments:
  - env: prod
manifestsValues: smelt/values.yaml # nexus-manifests file CI writes the image tag to
```

The `ApplicationSet` generates one `Application` per entry of `environments`, so a chart's whole
deployment is described next to the chart instead of in a central list. `prod` keeps the bare `app`
name and namespace; any other environment gets a `-<env>` suffix on both, so adding one never
touches what prod runs. The remaining keys (`namespace`, per-environment `valueFiles`,
`serverSideApply`, `deleteResources`) are described in the schema; the template reads nothing else,
and the repositories and the cluster are fixed in it. The same file is meant to grow with the
platform: new environments, and the promotion pipeline between them, derive from it.

Auto-sync is the one setting the UI owns: turning it off on an `Application` sticks, because the
`ApplicationSet` ignores that field once the `Application` exists. It is the incident switch, and a
paused app shows as such in the UI instead of hiding in a git diff.

Removing an `argo.yaml`, or an entry of its `environments`, deletes its `Application`, and with it
everything the app deployed unless `deleteResources` is `false`. Changing `app` is not a rename
either: the old `Application` is deleted and a new one created, so a stateful app loses its data.
Deleting the `ApplicationSet` itself deletes nothing (`preserveResourcesOnDeletion`): one bad edit
of app-of-apps can't take the whole platform down.

### Projects

Every `Application` belongs to one of two
<a href="https://argo-cd.readthedocs.io/en/stable/user-guide/projects/" target="_blank" rel="noopener"><code>AppProject</code>s</a>:

| Project    | Holds                       | Permissions                                                               |
| ---------- | --------------------------- | ------------------------------------------------------------------------- |
| `platform` | Every chart outside `apps/` | Any namespace, any resource, cluster-scoped included                      |
| `product`  | Charts under `apps/`        | Only its listed namespaces, no cluster-scoped resource, whitelisted kinds |

`argo.yaml` names its project, but the API server enforces the pairing: a
<a href="https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/" target="_blank" rel="noopener"><code>ValidatingAdmissionPolicy</code></a>
rejects any `Application` whose chart is under `apps/` and whose project is not `product`, or the
reverse. A wrong project fails loudly at sync instead of silently granting a product chart the
platform's permissions, which an `AppProject` alone can't prevent since it restricts repositories,
not paths. The same policy forbids moving an existing `Application` into `product`: an `argo.yaml`
reusing a platform app's name updates that `Application` instead of creating a new one, and would
otherwise take it over. A CEL policy was chosen over an admission webhook because it runs inside the
API server, with no server or certificate to operate.

The `product` whitelist opens whole API groups (core, workloads, Traefik, cert-manager, CNPG, ...)
and a blacklist removes `ResourceQuota` and `LimitRange`, so an app cannot lift its own namespace
limits. A kind outside the whitelist fails to sync and is also hidden from the app's resource tree,
even when an operator created it: that is why the whole core group is open, or the Pods behind a
`Deployment` would not show. A product chart that needs another group fails to sync until the group
is added to the whitelist.

**Runbook — add a new cluster-side workload:** add an `argo.yaml` next to its chart and push to
`main`. A product app under `apps/` also needs its namespace added to the `product` project's
`destinations`, the reviewed change that keeps an app out of `kube-system`. No one ever clicks
"create application" in the UI.

Every generated `Application` carries the
<a href="https://argo-cd.readthedocs.io/en/stable/user-guide/app_deletion/#about-the-deletion-finalizer" target="_blank" rel="noopener">resources
finalizer</a> unless its `argo.yaml` sets `deleteResources: false`, so deleting it cascades: ArgoCD
deletes everything the app deployed before the `Application` itself, instead of orphaning it. It is
the foreground variant, so a deletion stuck on a resource stays visible in the UI rather than
failing silently. Keep a product app's namespace in the `product` `destinations` until its deletion
completes. Deleting a stateful app's `Application` deletes its data with it.

## Sync model

| Workload             | Source of truth                             | Why                                                                |
| -------------------- | ------------------------------------------- | ------------------------------------------------------------------ |
| Platform components  | This repo (single-source `Application`)     | Manifests _are_ the source of truth — converge on every Git change |
| Apps shipping images | This repo (chart) + `nexus-manifests` (tag) | Image tag is decoupled from the chart so CI can bump it atomically |

Image-shipping apps use a
<a href="https://argo-cd.readthedocs.io/en/stable/user-guide/multiple_sources/" target="_blank" rel="noopener">multi-source
<code>Application</code></a>: the chart comes from this repo, and the values overlay its `argo.yaml`
names in `manifestsValues` comes from
<a href="https://github.com/kbntx-org/nexus-manifests" target="_blank" rel="noopener"><code>nexus-manifests</code></a>,
a small repo CI commits image-tag bumps to (see [GitOps deploys](03-gitops-deploys.md)). A GitHub
webhook triggers a sync within seconds of every `nexus-manifests` push.

## Dependency updates

<a href="https://docs.renovatebot.com/" target="_blank" rel="noopener">Renovate</a> runs self-hosted
on a scheduled GitHub Actions workflow rather than the hosted GitHub app. It authenticates with a
dedicated GitHub App (token minted per run via
<a href="https://github.com/actions/create-github-app-token" target="_blank" rel="noopener"><code>create-github-app-token</code></a>)
instead of the default `GITHUB_TOKEN`.

It runs every day at 04:00 UTC and only proposes releases at least 3 days old (`minimumReleaseAge`),
which leaves time for a broken or malicious release to be pulled. Most minor and patch updates
auto-merge once checks pass; majors, `0.x` minors and k3s wait for a human.

mise (`minimum_release_age` in `mise.toml`) and pnpm (`minimumReleaseAge` in `pnpm-workspace.yaml`)
apply the same 3-day age to manual installs, so a hand-added dependency cannot skip it either.

The job installs <a href="https://mise.jdx.dev/" target="_blank" rel="noopener">mise</a>-managed
tools via the same
<a href="https://github.com/kbntx-org/nexus/blob/main/.github/actions/mise-install/action.yaml" target="_blank" rel="noopener"><code>mise-install</code></a>
action as every other pipeline before invoking Renovate, so `postUpgradeTasks` like the pnpm
lockfile fixup run with the toolchain versions this repo pins, not whatever ships in a Renovate base
image.

## Access

Humans authenticate through GitHub SSO via the bundled
<a href="https://dexidp.io/" target="_blank" rel="noopener">Dex</a> connector. The local `admin`
account is disabled and RBAC maps a GitHub team to `role:admin`, so granting access is a
team-membership change, not an ArgoCD config edit.

The pipeline never talks to the ArgoCD API: a deploy is a commit to `nexus-manifests`, so CI needs
no ArgoCD credentials. The narrowly scoped `ci` API-key account still declared in the ArgoCD values
is not used by any workflow.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/argocd" target="_blank" rel="noopener"><code>platform/core/argocd/</code></a>
  — ArgoCD Helm chart wrapper, ingress, SSO + RBAC config
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/services/app-of-apps" target="_blank" rel="noopener"><code>platform/services/app-of-apps/</code></a>
  — root chart declaring the projects, the admission policy and the `applications` `ApplicationSet`
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/app-of-apps/values.yaml" target="_blank" rel="noopener"><code>platform/services/app-of-apps/values.yaml</code></a>
  — the `applications` `ApplicationSet` template
- <a href="https://github.com/kbntx-org/nexus/blob/main/.yaml-schemas/argo.schema.json" target="_blank" rel="noopener"><code>.yaml-schemas/argo.schema.json</code></a>
  — the keys an `argo.yaml` can set
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/app-of-apps/templates/validating-admission-policy.yaml" target="_blank" rel="noopener"><code>platform/services/app-of-apps/templates/validating-admission-policy.yaml</code></a>
  — ties `product` to `apps/` charts and stops apps from moving into it
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/renovate.yml" target="_blank" rel="noopener"><code>.github/workflows/renovate.yml</code></a>
  — scheduled Renovate pipeline
- <a href="https://github.com/kbntx-org/nexus/blob/main/renovate.json" target="_blank" rel="noopener"><code>renovate.json</code></a>
  — Renovate config
- [CI/CD pipeline](02-ci-cd-pipeline.md) — how source changes turn into images
- [GitOps deploys](03-gitops-deploys.md) — the multi-source `Application` shape and the
  `nexus-manifests` flow
