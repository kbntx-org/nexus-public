---
title: CI/CD pipeline
---

CI/CD runs on <a href="https://docs.github.com/en/actions" target="_blank" rel="noopener">GitHub
Actions</a>, colocated with the source.

## Runners

Every workflow runs on GitHub-hosted `ubuntu-latest` runners (`runs-on: ubuntu-latest`). No pipeline
step needs to reach the cluster, Vault, or any other internal network directly: images are built and
pushed to the registry, and a deploy is a commit to `nexus-manifests` — ArgoCD is what actually
reconciles the cluster from that repo, not this pipeline (see
[GitOps deploys](03-gitops-deploys.md)). Since nothing in the pipeline needs privileged network
access or in-cluster compute, hosted runners avoid the operational cost of running and isolating
CI's own compute (an ephemeral-pod controller, dedicated node pools, network policies) for no
corresponding benefit.

## Toolchain provisioning

Each job installs exactly the
<a href="https://mise.jdx.dev/" target="_blank" rel="noopener">mise</a>-tracked tools it needs — via
the
<a href="https://github.com/kbntx-org/nexus/blob/main/.github/actions/mise-install/action.yaml" target="_blank" rel="noopener"><code>mise-install</code></a>
composite action right after checkout — instead of running on a pre-baked image, listing only the
tools that job's steps actually invoke (e.g. `node pnpm` for a plain Nx job,
`node pnpm go golangci-lint` for a job whose `lint` target shells out to both). Versions come from
the root
<a href="https://github.com/kbntx-org/nexus/blob/main/mise.toml" target="_blank" rel="noopener"><code>mise.toml</code></a>,
the single source of truth for every pinned runtime — the same file a contributor's shell reads
locally (see [Local Development](../../getting-started/02-local-development.md)).

## Pipeline shape

<a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/checks.yml" target="_blank" rel="noopener"><code>checks.yml</code></a>
is the single entrypoint for both pull requests and pushes to `main` — each called workflow branches
on `github.event_name` internally where behavior needs to differ, rather than living as a separate
PR/main file.

```mermaid
%%{init: {'theme':'dark'}}%%
graph LR
    Push[push / PR] --> Affected[Affected]
    Affected --> Lint[Lint & format]
    Affected --> Test
    Affected --> Build[Build<br/>images, parallel]
    Lint & Test & Build --> Gate[Checks gate]
    Gate --> Deploy[Deploy<br/>gated on targets != empty]
    Gate & Deploy --> PGate[Pipeline gate]
```

0. **Affected** — a single job at the front of the pipeline that resolves what changed once (see
   [Affected detection](#affected-detection) below) and hands the result to every job after it.
1. **Lint & format** — `nx run-many` scoped to the projects Affected marked as lint-affected.
   Skipped entirely, runner and all, when that list is empty — both on a PR and on `main`.
2. **Test** — same scoping/skip behavior against the test-affected list.
3. **Build** — build — and, outside a PR, push — one image per deploy target, as parallel steps in
   one job. Skipped as a whole job when Affected found no deploy targets at all. On a PR the images
   are built but never pushed, so a broken Dockerfile fails the check without publishing anything.
4. **Checks gate** — reads the result of Affected, Lint & format, Test, and Build, and fails unless
   every one of them is `success` or `skipped`. This is what branch protection should require
   instead of the individual jobs — a job that's legitimately skipped (nothing affected) shouldn't
   be able to block a merge.
5. **Deploy** — only runs if Checks gate passed and there's at least one deploy target; see
   [GitOps deploys](03-gitops-deploys.md) for the full mechanics.
6. **Pipeline gate** — a second, non-required gate after Deploy, folding its result in too. It
   exists purely so [Affected detection](#diff-base) has one reliable "did this commit's pipeline,
   deploy included, actually complete" signal — branch protection stays on Checks gate, since a
   flaky PR-preview push shouldn't be able to block a merge.

## Affected detection

<a href="https://nx.dev/" target="_blank" rel="noopener">Nx</a> answers "what changed since
`<base>`", and every question the pipeline asks it — what to lint, what to test, what to build for
CI — is the exact same call shape:
`nx show projects --affected --base <base> --with-target <target>`. Each deployable project declares
a real `build-ci`
<a href="https://nx.dev/reference/project-configuration#targets" target="_blank" rel="noopener">Nx
target</a> in its `project.json`, deliberately named differently from the plain `build` target some
of these projects already have for local dev (`portfolio:build` is what `nx serve` depends on;
folding a Docker push into it would make local dev accidentally publish images). `build-ci` is an
`nx:run-commands` target that runs
<a href="https://github.com/kbntx-org/nexus/blob/main/tools/docker-build-and-push.sh" target="_blank" rel="noopener"><code>tools/docker-build-and-push.sh</code></a>.
Only projects that ship an image are in the Nx graph at all — pure infrastructure like the
[bastion](../traffic/01-overview.md#private-access-via-warp) has no `project.json` and is rolled out
by Terraform, not by this pipeline. All of this runs as steps directly inside the
<a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/affected.yml" target="_blank" rel="noopener"><code>Affected</code></a>
job — it has no read dependency on `nexus-manifests` at all; every app's deploy-target decision is
Nx's own affected-graph computation, diffed against the same shared base described below. `build-ci`
only produces and pushes the image; bumping the tag in `nexus-manifests` is still entirely
`deploy.yml`'s job, unchanged — see [GitOps deploys](03-gitops-deploys.md).

There's no hand-rolled path-matching or fail-safe script anymore — both are just Nx `inputs`.
`affected.yml`, `build.yml`, and `deploy.yml` are listed in `nx.json`'s `sharedGlobals` named input,
which every project's `default` (and therefore `production`, and therefore every target built on top
of it — `build`, `build-ci`, `test`, `lint`) already includes. So a change to any of those three
files changes every project's task hash for every target, and Nx's own affected computation marks
everything affected on its own — deliberately broader than the old fail-safe, which only ever
touched deploy targets; a pipeline-critical change now also re-lints and re-tests everything, not
just re-deploys it.

### Diff base

On a PR, the diff base is always trunk (`origin/main`), regardless of the PR's actual base branch —
this matters for stacked PRs targeting another feature branch.

On `main`, the diff base is **the most recent ancestor commit whose `Pipeline gate` job succeeded**,
not simply the previous commit. A commit can land on `main` without its pipeline ever going green —
an infra blip, a flaky job, a force-push, or a deploy that failed to push to `nexus-manifests` after
Checks gate already passed — and diffing against an unvalidated (or undeployed) commit would
silently drop whatever never actually shipped. Checking `Pipeline gate` specifically, rather than
`Checks gate`, is what makes that self-healing: since Pipeline gate also folds in Deploy's result, a
commit whose deploy failed is ineligible as a future diff base, so the very next run's diff
naturally widens to pick that change back up. So instead the action walks `Checks` runs on `main`
newest-first via the GitHub API, skipping any whose commit isn't a real ancestor of `HEAD` (guards
against two runs finishing out of order), and falls back to the empty tree — treating every project
as affected — logging a warning if no green ancestor is found at all.

This only catches drift `nexus`'s own pipeline can see, though — a manual `git revert` made directly
in `nexus-manifests` (see [Rollback and hotfix](03-gitops-deploys.md#rollback-and-hotfix)) has no
trace anywhere in `nexus`, so there's nothing for this walk to detect; re-syncing after a manual
rollback is a deliberate follow-up step, not something this makes automatic.

```mermaid
%%{init: {'theme':'dark'}}%%
graph TD
    Start[Walk Checks runs<br/>on main, newest first] --> Ancestor{commit is ancestor<br/>of HEAD?}
    Ancestor -->|no| Start
    Ancestor -->|yes| Gate{Pipeline gate<br/>succeeded?}
    Gate -->|no| Start
    Gate -->|yes| Base[Diff base = that commit]
    Start -->|runs exhausted| Empty[Diff base = empty tree<br/>+ warning logged]
```

Every app uses this same base — there's no separate per-app diff base read from `nexus-manifests`
anymore. The tradeoff: if two commits land on `main` close enough together that the second one's
diff base walk doesn't yet see the first as green, both can independently decide the same app is a
deploy target and each push their own tag. That's harmless, not incorrect — see
[Deploy target computation](03-gitops-deploys.md#deploy-target-computation) for why the write side
already makes the newer commit's tag win.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/actions/mise-install/action.yaml" target="_blank" rel="noopener"><code>.github/actions/mise-install/action.yaml</code></a>
  — per-job toolchain provisioning
- <a href="https://github.com/kbntx-org/nexus/blob/main/mise.toml" target="_blank" rel="noopener"><code>mise.toml</code></a>
  — pinned runtime versions
- <a href="https://github.com/kbntx-org/nexus/tree/main/.github/workflows" target="_blank" rel="noopener"><code>.github/workflows/</code></a>
  — workflow definitions
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/checks.yml" target="_blank" rel="noopener"><code>.github/workflows/checks.yml</code></a>
  — entrypoint wiring Affected, Lint & format, Test, Build, Checks gate, Deploy, and Pipeline gate
  together
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/affected.yml" target="_blank" rel="noopener"><code>.github/workflows/affected.yml</code></a>
  — the single upfront job: affected + deploy-target computation, including the trunk diff-base walk
