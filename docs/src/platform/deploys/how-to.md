---
title: Deploys
---

Merging to `main` deploys every affected app that ships an image: CI builds it and commits its tag
to `nexus-manifests`, which ArgoCD syncs. The Deploy job's summary lists each app, its values file
and the tag.

## Roll back

An image: pick one, in `nexus-manifests`, and ArgoCD syncs it within a few minutes (**Refresh** in
the UI to sync now):

- `git revert` the bad commit.
- Set `image.tag` in the app's file to a known-good tag and push. The registry keeps only the last
  few tags of each image; an older one needs a rebuild.

Then fix `main`: the next deploy of that app overwrites the manual tag.

A config change: revert its PR in `nexus`. Rolling back from the ArgoCD UI history doesn't stick:
auto-sync re-applies `main` unless it is paused.

## Redeploy the latest commit

If an app runs an older tag than `main` should (two close commits deployed out of order), re-run the
Deploy job of the newest commit.

How it works: [Deploys design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus-manifests" target="_blank" rel="noopener"><code>nexus-manifests</code></a>
  — one values file per app
