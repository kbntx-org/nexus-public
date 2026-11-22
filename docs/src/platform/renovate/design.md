---
title: Renovate design
---

<a href="https://docs.renovatebot.com/" target="_blank" rel="noopener">Renovate</a> opens a PR for
every version pinned in the repository: the usual manifests, plus what the `customManagers` in
`renovate.json` find elsewhere.

## How it's set up

It runs self-hosted, daily, from a scheduled GitHub Actions workflow, with a token from a dedicated
GitHub App minted per run. It installs the repository's mise tools first, so post-upgrade tasks (the
pnpm lockfile fix-up) run with our pinned versions, not a Renovate image's.

## Decisions and gotchas

- **Self-hosted rather than the hosted app**, to control the schedule, the toolchain and the token.
- **Releases must be at least 3 days old**, which leaves time for a broken or malicious release to
  be pulled. mise and pnpm apply the same age to manual installs.
- **Minor and patch updates auto-merge** once checks pass; majors and `0.x` minors wait for a
  review.
- **k3s minors are disabled and patches need a review** (see
  [Upgrade k3s](../kubernetes/how-to.md#upgrade-k3s)).
- **Related versions are grouped** into one PR, such as an umbrella chart's sub-charts, or the
  OpenTofu version in `mise.toml` and the stacks' `required_version`.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/renovate.json" target="_blank" rel="noopener"><code>renovate.json</code></a>
  — rules, groups and custom managers
- <a href="https://github.com/kbntx-org/nexus/blob/main/.github/workflows/renovate.yml" target="_blank" rel="noopener"><code>.github/workflows/renovate.yml</code></a>
  — the scheduled run
