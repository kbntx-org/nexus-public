---
title: Renovate
---

Renovate opens dependency update PRs every day; most minor and patch updates merge themselves.

## Review an update that waits

Majors, `0.x` minors and k3s patches need a review: read the release notes, check CI and merge.
Renovate rebases its PRs only on conflict.

## Make a new pinned version updatable

Renovate finds versions in the usual manifests. A version pinned anywhere else (a `Dockerfile`
`ARG`, a Tiltfile) needs a `# renovate: datasource=... depName=...` comment, and its file added to
the `fileMatch` of the matching `customManagers` entry (or a new manager) in
<a href="https://github.com/kbntx-org/nexus/blob/main/renovate.json" target="_blank" rel="noopener"><code>renovate.json</code></a>.

## Run it now

Run the **Renovate** workflow from the Actions tab.

How it works: [Renovate design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/renovate.json" target="_blank" rel="noopener"><code>renovate.json</code></a>
  — config
