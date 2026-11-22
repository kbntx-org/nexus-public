---
title: Deploys design
---

An app that ships an image is deployed by committing its new tag to
<a href="https://github.com/kbntx-org/nexus-manifests" target="_blank" rel="noopener"><code>nexus-manifests</code></a>,
a small repository ArgoCD reads as a values overlay. It holds one tiny file per app
(`kiln/values.yaml` → `image: { tag: <tag> }`), no chart and no templates. Its history is the deploy
log.

## How it's set up

1. **Build** tags each image `trunk-<sha>` on `main`, `pr-<number>-commit-<sha>` on a PR.
2. **Deploy** (`deploy.yml`, one job) clones `nexus-manifests`, bumps `image.tag` in the file each
   target's `argo.yaml` names in `manifestsValues`, and pushes one commit per run. A target without
   `manifestsValues`, such as a base image, is built but not deployed.
3. **ArgoCD** picks up the new commit on its next poll, within a few minutes.

A `nexus-ci` GitHub App mints a short-lived token for that push, the only write CI makes to
`nexus-manifests`.

## Decisions and gotchas

- **A manifests repository rather than `argocd app set`**: a live override is unversioned, needs CI
  credentials and network access to ArgoCD, two close builds can interleave, and `selfHeal` reverts
  it anyway. A commit is versioned, serialized and reverted with `git revert`.
- **Deploys to a branch run one at a time** (concurrency group `deploy-<branch>`). On `main` they
  queue first in, first out; on a PR branch a newer deploy replaces the pending one. A push that
  still races retries with a rebase.
- **Two close commits can deploy the same app.** The second's diff base doesn't see the first as
  shipped yet. The queue keeps commit order, but it doesn't compare tags: if the older pipeline
  reaches Deploy last, its older tag stays live.
- **CI never talks to the ArgoCD API**, so it holds no ArgoCD credentials.
- **A manual change in `nexus-manifests` is invisible to CI**, so nothing undoes or detects a
  hand-made rollback.
- **PRs build without pushing** and write their tag to a `pr-<number>` branch, deleted when the PR
  closes. It is scaffolding for PR previews, not consumed yet.
- **The bastion is not deployed this way**: it is a VM managed by OpenTofu, not a cluster workload.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/deploy.yml" target="_blank" rel="noopener"><code>deploy.yml</code></a>
  — the `nexus-manifests` bump
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/cleanup-pr-manifests.yml" target="_blank" rel="noopener"><code>cleanup-pr-manifests.yml</code></a>
  — deletes a PR's branch on close
