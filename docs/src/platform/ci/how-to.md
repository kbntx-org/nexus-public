---
title: CI
---

CI checks every pull request and push to `main` and builds and deploys what changed. The required
check is **Checks gate**.

## Run the CI checks locally

Before every commit, run what CI runs on your change:

```sh
pnpm nx format:check --base main --libs-and-apps
pnpm nx format:check --base main
pnpm nx affected -t format-check test
```

Both `format:check` runs are needed: the first covers files in an Nx project, the second the rest of
the diff.

## Make a project build and deploy

1. Give it a `build-ci` target that runs `tools/docker-build-and-push.sh` (copy
   <a href="https://github.com/kbntx-org/nexus/blob/main/docs/project.json" target="_blank" rel="noopener"><code>docs/project.json</code></a>).
2. Set its `toolchain` named input to the toolchains it builds with (`goToolchain`,
   `nodeToolchain`).
3. If its image bundles another project, list that project in `implicitDependencies`.
4. Add an `argo.yaml` with `manifestsValues` next to its chart, under the project's root (the deploy
   job looks nowhere else), or CI builds the image but deploys nothing (see
   [Add an app](../argocd/how-to.md#add-an-app)).

## Run CI on a draft PR

Draft PRs run nothing, to save CI minutes. Mark the PR ready for review, or `gh stack submit --open`
for a stack.

## Fix a commit that didn't ship

Nothing to do: the next push to `main` diffs against the last commit that fully shipped, so it
rebuilds and redeploys what was missed. To ship it now, re-run the failed jobs of that commit.

How it works: [CI design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/checks.yml" target="_blank" rel="noopener"><code>checks.yml</code></a>
  — the pipeline
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/actions/mise-install/action.yaml" target="_blank" rel="noopener"><code>mise-install</code></a>
  — per-job tool install
