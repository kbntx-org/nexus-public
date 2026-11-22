---
title: ArgoCD
---

ArgoCD deploys every chart that has an `argo.yaml`. Its UI, behind GitHub login and WARP, shows each
app's sync status, health and resources.

## Add an app

Add an `argo.yaml` next to the chart's `Chart.yaml` and merge. Nobody creates an `Application` in
the UI.

```yaml
# yaml-language-server: $schema=../../../.yaml-schemas/argo.schema.json
app: my-app # unique across the repository
project: platform # product under apps/, platform anywhere else
environments:
  - env: prod
manifestsValues: my-app/values.yaml # only for an app CI builds an image for; created on first deploy
```

A product app under `apps/` also needs its namespace in the `product` project's `destinations`, and
a kind outside its whitelist needs its API group added, both in
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/app-of-apps/values.yaml" target="_blank" rel="noopener"><code>app-of-apps/values.yaml</code></a>.
The other keys (`namespace`, `valueFiles`, `autoSync`, `serverSideApply`, `deleteResources`) are
described in the schema.

## Remove or rename an app

Removing its `argo.yaml` (or an environment) deletes the app and everything it deployed. Renaming
`app` deletes it and creates a new one. For a stateful app, set `deleteResources: false` first and
merge that, or back up its data. Keep a product app's namespace in `destinations` until the deletion
has finished.

## Pause auto-sync during an incident

In the app's details, open the sync policy and disable auto-sync. It stays off until you turn it
back on. While it's off, the app shows on the **Auto-sync overridden** panel of the **Argo CD -
Applications** dashboard: switch it back when done. Changing `autoSync` in `argo.yaml` for an
existing app only changes the `nexus/auto-sync` label the panel compares against; the live setting
still has to be switched in the UI to match.

## Debug a sync

- **Sync fails with a permission error on a product app:** its namespace isn't in `destinations`, or
  the kind isn't whitelisted.
- **Sync fails with "metadata.annotations: Too long":** set `serverSideApply: true`.
- **A deletion hangs:** a resource is stuck on its own finalizer; it shows in the app's resource
  tree.

How it works: [ArgoCD design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/.yaml-schemas/argo.schema.json" target="_blank" rel="noopener"><code>argo.schema.json</code></a>
  — `argo.yaml` keys
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/argocd/dashboards" target="_blank" rel="noopener"><code>platform/core/argocd/dashboards/</code></a>
  — the Argo CD - Applications dashboard
