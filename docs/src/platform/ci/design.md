---
title: CI design
---

CI runs on <a href="https://docs.github.com/en/actions" target="_blank" rel="noopener">GitHub
Actions</a>, on GitHub-hosted runners, and only checks, builds and deploys what a change affects, as
<a href="https://nx.dev/" target="_blank" rel="noopener">Nx</a> computes it.

## How it's set up

`checks.yml` is the single entrypoint for pull requests and pushes to `main`; each called workflow
branches on the event where behavior differs.

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/pipeline.svg"
</div>

0. **Affected** resolves once what changed and hands each job its list of projects.
1. **Lint & format** always runs the oxfmt check, then the affected `lint` and `format-check`
   targets.
2. **Test** runs the affected `test` targets.
3. **Build** runs the affected `build-ci` targets, one image per deploy target, and pushes them to
   the [container registry](../container-registry/design.md) outside a PR.
4. **Checks gate** fails unless every job above succeeded or was skipped. It is the required check,
   so a job skipped because nothing is affected never blocks a merge.
5. **Deploy** bumps the image tags in `nexus-manifests` (see [Deploys](../deploys/design.md)).
6. **Pipeline gate**, on `main` only, folds Deploy into Checks gate: it marks a commit as fully
   shipped.

## Diff base

On a PR, Nx diffs against `origin/main`, even for a stacked PR. On `main`, it diffs against the
latest ancestor whose **Pipeline gate** succeeded, not the previous commit. Runs on `main` are never
cancelled (their concurrency group includes the commit), so every trunk commit leaves a result:

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/diff-base.svg"
</div>

A commit can land without shipping (a flaky job, a failed deploy). Diffing against it would drop
what never shipped; skipping it widens the next diff to pick that change back up. With no green
ancestor, every project is affected.

## Decisions and gotchas

- **Hosted runners**: the only internal access CI needs is a WARP service-token connection to push
  to zot, and a deploy is a Git commit, so self-hosted runners would add compute to run and isolate
  for no benefit. The build job disconnects WARP even when it fails, since a registration left
  behind stays in the Zero Trust device list forever.
- **Affected is only Nx `inputs`**, no path matching. A workflow change affects nothing: it changes
  how CI runs, not what a target builds, and its own PR already runs the new version.
- **Toolchains are scoped per project.** Each project opts into its toolchain file in
  `.config/mise/conf.d/` through its `toolchain` named input, so a Go bump rebuilds only Go
  projects. An Nx `runtime` input wouldn't do: it changes the cache hash, not `affected`.
- **npm bumps affect only their dependents**: `projectsAffectedByDependencyUpdates: auto` diffs the
  lockfile and walks the project graph. When Nx can't map a change, every project is affected.
- **A project whose image bundles another lists it in `implicitDependencies`**, since Nx can't read
  a Dockerfile.
- **`build-ci`, not `build`, makes a deploy target**, so local dev can never publish an image.
- **Each job installs only the tools it runs** through the `mise-install` action, at the versions
  your shell uses.
- **A public mirror** is force-pushed daily by `sync-nexus-public.yml`, with excluded paths and
  personal emails stripped from the whole history by
  <a href="https://github.com/newren/git-filter-repo" target="_blank" rel="noopener">git-filter-repo</a>,
  since an old commit would still expose them. Never edit the mirror.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/.github/workflows" target="_blank" rel="noopener"><code>.github/workflows/</code></a>
  — `checks.yml` and the workflows it calls
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/affected.yml" target="_blank" rel="noopener"><code>affected.yml</code></a>
  — affected projects and the diff-base walk
- <a href="https://github.com/kbntx-org/nexus/blob/main/nx.json" target="_blank" rel="noopener"><code>nx.json</code></a>
  — toolchain named inputs and lockfile mode
- <a href="https://github.com/kbntx-org/nexus/blob/main/tools/docker-build-and-push.sh" target="_blank" rel="noopener"><code>tools/docker-build-and-push.sh</code></a>
  — the script every `build-ci` target runs
