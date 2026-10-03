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
<a href="https://github.com/argoproj/argo-helm/tree/main/charts/argocd-apps" target="_blank" rel="noopener"><code>argocd-apps</code></a>
that declares every other `Application` the platform needs as a child. Bootstrapping the platform is
one sync of the root; everything declared in
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/app-of-apps/values.yaml" target="_blank" rel="noopener"><code>app-of-apps/values.yaml</code></a>
is then materialized on its own: every `platform/core/*` and `platform/services/*` component plus
the apps, all traceable back to one file.

**Runbook — add a new cluster-side workload:** drop the chart under `platform/core/<name>/` or
`platform/services/<name>/`, add an entry under `argocd-apps.applications` in
`app-of-apps/values.yaml`, push to `main`. No one ever clicks "create application" in the UI.

## Sync model

| Workload             | Source of truth                             | Why                                                                |
| -------------------- | ------------------------------------------- | ------------------------------------------------------------------ |
| Platform components  | This repo (single-source `Application`)     | Manifests _are_ the source of truth — converge on every Git change |
| Apps shipping images | This repo (chart) + `nexus-manifests` (tag) | Image tag is decoupled from the chart so CI can bump it atomically |

Image-shipping apps use a
<a href="https://argo-cd.readthedocs.io/en/stable/user-guide/multiple_sources/" target="_blank" rel="noopener">multi-source
<code>Application</code></a>: the chart comes from this repo, and a values overlay
(`$values/<app>/values.yaml`) comes from
<a href="https://github.com/kbntx-org/nexus-manifests" target="_blank" rel="noopener"><code>nexus-manifests</code></a>,
a small repo CI commits image-tag bumps to (see [GitOps deploys](03-gitops-deploys.md)). A GitHub
webhook triggers a sync within seconds of every `nexus-manifests` push.

## Dependency updates

<a href="https://docs.renovatebot.com/" target="_blank" rel="noopener">Renovate</a> runs self-hosted
on a scheduled GitHub Actions workflow rather than the hosted GitHub app. It authenticates with a
dedicated GitHub App (token minted per run via
<a href="https://github.com/actions/create-github-app-token" target="_blank" rel="noopener"><code>create-github-app-token</code></a>)
instead of the default `GITHUB_TOKEN`.

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
  — root chart declaring every child `Application`
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/app-of-apps/values.yaml" target="_blank" rel="noopener"><code>platform/services/app-of-apps/values.yaml</code></a>
  — the full catalog of cluster workloads
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/renovate.yml" target="_blank" rel="noopener"><code>.github/workflows/renovate.yml</code></a>
  — scheduled Renovate pipeline
- <a href="https://github.com/kbntx-org/nexus/blob/main/renovate.json" target="_blank" rel="noopener"><code>renovate.json</code></a>
  — Renovate config
- [CI/CD pipeline](02-ci-cd-pipeline.md) — how source changes turn into images
- [GitOps deploys](03-gitops-deploys.md) — the multi-source `Application` shape and the
  `nexus-manifests` flow
