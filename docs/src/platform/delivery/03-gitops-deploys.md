---
title: GitOps deploys
---

Apps that ship an image are deployed by committing their new tag to a **separate manifests repo**
that ArgoCD reads; CI never patches ArgoCD. This page covers why, how the deploy job writes to that
repo, and how to roll back.

## Why a separate repo

A deploy has to change the image tag ArgoCD runs. The alternative to a manifests repo is CI mutating
the live `Application` (`argocd app set` with the new tag): that change is unversioned, needs CI
credentials and network reach into ArgoCD, and two builds landing close together can interleave
their calls and leave an older image live after a newer commit has "deployed". It also fights
`selfHeal: true`, which reverts any live override to what Git declares on the next reconcile, so
`argocd app set` is never part of deploying or rolling back.

Instead, every deploy is a commit to a small, dumb git repo,
<a href="https://github.com/kbntx-org/nexus-manifests" target="_blank" rel="noopener"><code>nexus-manifests</code></a>,
that ArgoCD reads as a values overlay (see [Sync model](01-overview.md#sync-model)). It holds one
tiny file per app (`portfolio/values.yaml` → `image: { tag: <tag> }`, for example), no `Chart.yaml`,
no templates. Writes to it are serialized, its history _is_ the deploy log, and rollback is
`git revert` on this repo alone.

## Build and deploy

The chain is drawn in [Pipeline shape](02-ci-cd-pipeline.md#pipeline-shape).
<a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/build.yml" target="_blank" rel="noopener"><code>build.yml</code></a>
runs `pnpm nx run-many --target=build-ci --projects=<deploy targets>` with the image tag in an
`IMAGE_TAG` env var: `trunk-<commit sha>` on `main`, `pr-<number>-commit-<head sha>` on a PR.
`build-ci` only produces and pushes the image; bumping the tag in `nexus-manifests` is
`deploy.yml`'s job.

<a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/deploy.yml" target="_blank" rel="noopener"><code>deploy.yml</code></a>
is **one job** (`deploy`), invoked only through `workflow_call` once Checks gate has passed. It
clones `nexus-manifests`, bumps `image.tag` for each target and makes one commit per run:

- Runs share a concurrency group per target branch (`deploy-<branch>`), so only one deploy writes to
  a branch at a time.
- On `main`, waiting deploys queue first-in, first-out, so every trunk commit that reaches Deploy
  gets its turn. On a PR branch a newer deploy replaces the pending one, since only the latest
  preview matters.
- The push retries with a rebase, covering any write that still lands in between (a hand-made hotfix
  commit, for example).

ArgoCD auto-syncs the new commit within seconds.

## Which projects get a manifests bump

A `build-ci` target makes a project a deploy target, but a manifests bump also needs a
`manifestsValuesPath` in its `project.json` metadata: `deploy.yml` reads it per target and skips any
project without one. A standalone base image with no Kubernetes workload of its own (the
<a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/sysbox" target="_blank" rel="noopener">sysbox</a>
image, for example) is built and pushed like any other project but has nothing for `nexus-manifests`
to track.

## Concurrent deploys

Two commits landing on `main` close together can both deploy the same app: the second's
[diff base](02-ci-cd-pipeline.md#diff-base) walk doesn't yet see the first as green, so both pick
the app as a deploy target and each push a commit bumping its tag.

The deploy queue runs them in the order their pipelines reach Deploy, normally commit order, so the
newer tag lands last and the cost is one redundant image build. The queue doesn't compare tags,
though: if the older commit's pipeline reaches Deploy after the newer one, its older tag lands last
and stays live. If an app runs an older tag than `main` should have, re-run the newest commit's
Deploy job or hand-edit the tag (see [Rollback and hotfix](#rollback-and-hotfix)).

## Auth

A `nexus-ci` GitHub App, installed on both repos with write access to `nexus-manifests`, mints a
short-lived installation token for the deploy job's write, the only thing in CI that touches
`nexus-manifests`.

## Rollback and hotfix

The manifests repo is small and human-editable, so when production needs to move right now:

- `git revert` the bad commit in `nexus-manifests`; ArgoCD syncs the revert within seconds.
- Hand-edit `image.tag` to a known-good tag and push.
- Re-run CI for the desired `nexus` commit.

A failed deploy heals itself on the next run (see [Diff base](02-ci-cd-pipeline.md#diff-base)), but
a manual change in `nexus-manifests` leaves no trace in `nexus`, so nothing detects it: re-syncing
after a manual rollback is a deliberate follow-up, not automatic.

## Pull requests

PRs run the same build and deploy chain with two differences. The build never pushes the image
(`docker buildx build` without a registry output), so a broken Dockerfile still fails the check
without publishing anything. The deploy writes the `pr-<number>-commit-<head sha>` tag to a
`pr-<number>` branch in `nexus-manifests` instead of `main`, created from `main` on first build and
reused after.

Since the image is never pushed, that tag isn't consumable yet: it's scaffolding for a not-yet-wired
PR-preview `ApplicationSet`.
<a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/cleanup-pr-manifests.yml" target="_blank" rel="noopener"><code>cleanup-pr-manifests.yml</code></a>
deletes the `pr-<number>` branch when the PR closes, merged or not.

## What's not GitOps-managed

The
<a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/bastion" target="_blank" rel="noopener"><code>bastion</code></a>
is a VPS, not a cluster workload, and has no `project.json`, so neither ArgoCD nor this pipeline
deploys it; Terraform does. That keeps the one component that can't run in the cluster out of the
delivery pipeline instead of bolting a second deploy mechanism onto it. See
[private access](../traffic/01-overview.md#private-access-via-warp) for how it is provisioned and
why it exists.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/app-of-apps/values.yaml" target="_blank" rel="noopener"><code>platform/services/app-of-apps/values.yaml</code></a>
  — multi-source `Application` definitions for the image-shipping apps
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/checks.yml" target="_blank" rel="noopener"><code>.github/workflows/checks.yml</code></a>
  — entrypoint wiring affected/lint/test/build/deploy together
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/affected.yml" target="_blank" rel="noopener"><code>.github/workflows/affected.yml</code></a>
  — deploy-target computation, diffed against the shared trunk base
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/build.yml" target="_blank" rel="noopener"><code>.github/workflows/build.yml</code></a>
  — runs `nx run-many --target=build-ci` for the affected deploy targets
- <a href="https://github.com/kbntx-org/nexus/blob/main/tools/docker-build-and-push.sh" target="_blank" rel="noopener"><code>tools/docker-build-and-push.sh</code></a>
  — the shared build/push script every `build-ci` target runs
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/deploy.yml" target="_blank" rel="noopener"><code>.github/workflows/deploy.yml</code></a>
  — the single `deploy` job (the `nexus-manifests` bump)
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/cleanup-pr-manifests.yml" target="_blank" rel="noopener"><code>.github/workflows/cleanup-pr-manifests.yml</code></a>
  — deletes the PR branch on close
