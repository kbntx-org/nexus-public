---
title: CI/CD pipeline
---

CI/CD runs on <a href="https://docs.github.com/en/actions" target="_blank" rel="noopener">GitHub
Actions</a>, colocated with the source.

## Runners

Every workflow runs on GitHub-hosted `ubuntu-latest` runners. No step needs to reach the cluster,
Vault or any other internal network: images are pushed to the registry, and a deploy is a commit to
`nexus-manifests` that ArgoCD reconciles from (see [GitOps deploys](03-gitops-deploys.md)). With no
need for privileged network access or in-cluster compute, hosted runners spare us the cost of
running and isolating CI's own compute (an ephemeral-pod controller, dedicated node pools, network
policies) for no corresponding benefit.

## Toolchain provisioning

Each job installs only the
<a href="https://mise.jdx.dev/" target="_blank" rel="noopener">mise</a>-tracked tools its steps
invoke, through the
<a href="https://github.com/kbntx-org/nexus/blob/main/.github/actions/mise-install/action.yaml" target="_blank" rel="noopener"><code>mise-install</code></a>
composite action right after checkout, instead of running on a pre-baked image: `node pnpm` for a
plain Nx job, `node pnpm go golangci-lint` for a job whose `lint` target shells out to both.

Versions come from mise's config, the single source of truth for every pinned runtime and the same
config a contributor's shell reads locally: language toolchains in
<a href="https://github.com/kbntx-org/nexus/tree/main/.config/mise/conf.d" target="_blank" rel="noopener"><code>.config/mise/conf.d/</code></a>,
every other tool in the root
<a href="https://github.com/kbntx-org/nexus/blob/main/mise.toml" target="_blank" rel="noopener"><code>mise.toml</code></a>
(see [Local Development](../../getting-started/02-local-development.md)).

## Pipeline shape

<a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/checks.yml" target="_blank" rel="noopener"><code>checks.yml</code></a>
is the single entrypoint for pull requests and pushes to `main`; each called workflow branches on
`github.event_name` where behavior differs, rather than living in a separate PR or main file. A
newer push to a PR cancels that PR's in-flight run, while every push to `main` gets its own run that
is never cancelled, so each trunk commit leaves a complete result for the [diff base](#diff-base)
walk.

Draft PRs skip every job, to save CI minutes while a change is still in progress; marking the PR
ready for review triggers the full run. Checks gate is skipped on a draft, which branch protection
treats as passing, but GitHub won't merge a draft anyway. `gh stack submit --auto` opens PRs as
drafts, and `gh stack submit --open` marks the whole stack ready.

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/pipeline.svg"
</div>

0. **Affected** resolves what changed once and hands the result to every job after it (see
   [Affected detection](#affected-detection)).
1. **Lint & format** always runs. The oxfmt check (`nx format:check` against the affected base, with
   and without `--libs-and-apps`, since each mode alone misses some files) runs on every pipeline.
   The `lint` targets (oxlint for TypeScript projects, inferred by `@nx/oxlint`) and `format-check`
   targets (`format-check` is a project's non-oxfmt formatter, such as `golangci-lint fmt` for a Go
   project) run through `nx run-many` on the projects Affected computed for each, and each step is
   skipped when its list is empty.
2. **Test** runs `nx run-many` on the test-affected projects; the job is skipped when that list is
   empty.
3. **Build** builds one image per deploy target in one job and, outside a PR, pushes it to Docker
   Hub (see [Pull requests](03-gitops-deploys.md#pull-requests)). It is skipped when Affected found
   no deploy targets.
4. **Checks gate** fails unless Affected, Lint & format, Test and Build each ended `success` or
   `skipped`. Branch protection should require this job rather than the individual ones, so a job
   legitimately skipped because nothing is affected can't block a merge.
5. **Deploy** runs only if Checks gate passed and Affected found at least one deploy target (see
   [GitOps deploys](03-gitops-deploys.md#build-and-deploy)).
6. **Pipeline gate** runs on trunk pushes only, after Deploy, and folds Deploy's result in with
   Checks gate's. It is the [diff base](#diff-base) walk's signal that a commit fully shipped.
   Branch protection stays on Checks gate, since a flaky PR-preview push shouldn't block a merge.

## Affected detection

<a href="https://nx.dev/" target="_blank" rel="noopener">Nx</a> decides what's affected; every job
after it follows that answer. Each question the pipeline asks (what to lint, format-check, test or
build) is the same call: `nx show projects --affected --base <base> --with-target <target>`.

What makes a project affected is entirely Nx `inputs`; there's no hand-rolled path matching. A
project is affected when a file under its root changes, when a workspace file one of its target
inputs names changes, or when a project it depends on is affected. Nothing outside a project marks
every project affected: a workflow change alters how the pipeline runs, not what a target produces,
and the PR that edits a workflow already runs the new version.

Toolchains are scoped per project: every project's `default` input (and so `production`, and every
target built on it) includes a `toolchain` named input, empty in `nx.json` and overridden in a
project's `project.json` with the toolchains it builds with (`goToolchain`, `nodeToolchain`), each
pointing at its file in `.config/mise/conf.d/`. A Go bump then rebuilds only Go projects, and a bump
of a tool in the root `mise.toml` (Tilt, kind, Colima) rebuilds nothing that doesn't opt into it.
Splitting by file rather than using an Nx `runtime` input (`mise current go`) is deliberate: a
`runtime` input only changes the cache hash, and `affected` only sees changed files, so a Go bump
would never redeploy anything. The folder is `.config/mise/conf.d/` because mise loads every file
there with no `MISE_ENV` (a root `mise.go.toml` is an environment file, loaded only when
`MISE_ENV=go`), and Renovate's mise manager matches it by default. Its tools are locked in a
separate `.config/mise/mise.lock`.

npm packages are scoped the same way. `nx.json` sets the JS plugin's
`projectsAffectedByDependencyUpdates` to `auto`: Nx parses the pnpm lockfile at both revisions,
diffs the packages, and walks the project graph back from the changed ones, so a Vite bump affects
only the projects that depend on Vite, directly or transitively. A project depends on a package
through its `package.json` and its imports (`analyzePackageJson`, `analyzeSourceFiles`). When Nx
can't map a change (a removed package, an unreadable base revision), it falls back to marking every
project affected. A package only the root tooling uses (oxfmt) affects nothing.

A deploy target whose image bundles another project declares it in `implicitDependencies` (kiln on
`kiln-client`), since the graph can't see a Dockerfile; without it, a change to the bundled project
alone would never rebuild the image.

A deploy target is a project with a `build-ci`
<a href="https://nx.dev/reference/project-configuration#targets" target="_blank" rel="noopener">Nx
target</a>. It is named apart from the plain `build` target some projects have for local dev
(`cloudflare-controller:build` runs `go build`), so local dev can never publish an image. `build-ci`
is an `nx:run-commands` target running
<a href="https://github.com/kbntx-org/nexus/blob/main/tools/docker-build-and-push.sh" target="_blank" rel="noopener"><code>tools/docker-build-and-push.sh</code></a>,
which builds with Buildx and pushes only when the workflow sets `PUSH`.

All of this runs as steps inside the
<a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/affected.yml" target="_blank" rel="noopener"><code>Affected</code></a>
job, which never reads `nexus-manifests`: every app's deploy decision is Nx's affected graph against
the one shared diff base below, with no per-app base. Code without a `project.json` is outside the
graph and so outside this pipeline (see
[What's not GitOps-managed](03-gitops-deploys.md#whats-not-gitops-managed)).

### Diff base

On a PR, the diff base is always trunk (`origin/main`), whatever the PR's base branch, so a stacked
PR targeting another feature branch still diffs against trunk.

On `main`, the diff base is **the most recent ancestor commit whose `Pipeline gate` job succeeded**,
not the previous commit:

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/diff-base.svg"
</div>

A commit can land on `main` without its pipeline going green: an infra blip, a flaky job, a
force-push, or a deploy that failed to push to `nexus-manifests` after Checks gate passed. Diffing
against such a commit would silently drop whatever never shipped.

Checking Pipeline gate rather than Checks gate makes this self-healing. Pipeline gate folds in
Deploy's result, so a commit whose deploy failed can't become a diff base, and the next run's diff
widens to pick that change back up.

The walk reads `Checks` runs on `main` through the GitHub API. The ancestor check guards against two
runs finishing out of order. With no green ancestor at all, the base falls back to the empty tree:
every project is affected and a warning is logged.

Two limits live on the deploy side: a manual change in `nexus-manifests` is invisible to this walk
(see [Rollback and hotfix](03-gitops-deploys.md#rollback-and-hotfix)), and two close commits can
both deploy the same app (see [Concurrent deploys](03-gitops-deploys.md#concurrent-deploys)).

## Public mirror

<a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/sync-nexus-public.yml" target="_blank" rel="noopener"><code>sync-nexus-public.yml</code></a>
runs daily (and on demand) outside the Checks pipeline and force-pushes a read-only copy of this
repo to a separate public repository. It rewrites the full history with
<a href="https://github.com/newren/git-filter-repo" target="_blank" rel="noopener">git-filter-repo</a>
rather than copying a snapshot: excluded paths and personal emails are stripped from every commit,
not just the latest tree, since anything left in an old commit would still be public. Keeping the
working repo private and publishing a filtered mirror makes the platform public without exposing
what is excluded from it (a private app, for example). The mirror is regenerated and force-pushed
each run, so it is never edited directly.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/actions/mise-install/action.yaml" target="_blank" rel="noopener"><code>.github/actions/mise-install/action.yaml</code></a>
  — per-job toolchain provisioning
- <a href="https://github.com/kbntx-org/nexus/blob/main/mise.toml" target="_blank" rel="noopener"><code>mise.toml</code></a>
  — pinned versions of the tools no build depends on
- <a href="https://github.com/kbntx-org/nexus/tree/main/.config/mise/conf.d" target="_blank" rel="noopener"><code>.config/mise/conf.d/</code></a>
  — pinned language toolchains, one file per toolchain
- <a href="https://github.com/kbntx-org/nexus/tree/main/.github/workflows" target="_blank" rel="noopener"><code>.github/workflows/</code></a>
  — workflow definitions
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/checks.yml" target="_blank" rel="noopener"><code>.github/workflows/checks.yml</code></a>
  — entrypoint wiring Affected, Lint & format, Test, Build, Checks gate, Deploy, and Pipeline gate
  together
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/affected.yml" target="_blank" rel="noopener"><code>.github/workflows/affected.yml</code></a>
  — the single upfront job: affected + deploy-target computation, including the trunk diff-base walk
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/lint-and-format.yml" target="_blank" rel="noopener"><code>.github/workflows/lint-and-format.yml</code></a>
  — oxfmt check plus the affected `lint` and `format-check` targets
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/build.yml" target="_blank" rel="noopener"><code>.github/workflows/build.yml</code></a>
  — runs `build-ci` for the deploy targets, pushing outside PRs
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/actions/setup-buildx/action.yaml" target="_blank" rel="noopener"><code>.github/actions/setup-buildx/action.yaml</code></a>
  — Buildx builder used by the build job
- <a href="https://github.com/kbntx-org/nexus/blob/main/tools/docker-build-and-push.sh" target="_blank" rel="noopener"><code>tools/docker-build-and-push.sh</code></a>
  — the shared build/push script every `build-ci` target runs
- <a href="https://github.com/kbntx-org/nexus/blob/main/nx.json" target="_blank" rel="noopener"><code>nx.json</code></a>
  — toolchain named inputs, the JS plugin's lockfile mode and target input defaults
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/sync-nexus-public.yml" target="_blank" rel="noopener"><code>.github/workflows/sync-nexus-public.yml</code></a>
  — daily public mirror with filtered history
