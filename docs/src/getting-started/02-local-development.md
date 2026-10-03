---
title: Local Development
---

The local environment runs inside a real Kubernetes cluster
(<a href="https://kind.sigs.k8s.io/" target="_blank" rel="noopener">kind</a>), orchestrated by
<a href="https://tilt.dev/" target="_blank" rel="noopener">Tilt</a>, with the **same ingress
controller and Helm charts** as production. A parallel `docker compose` setup was rejected because
it drifts from the real thing; here a change is tested against the same primitives before it ships.

The only prerequisite is
<a href="https://docs.docker.com/engine/install/" target="_blank" rel="noopener">Docker</a>. Every
other tool is pinned in
<a href="https://github.com/kbntx-org/nexus/blob/main/mise.toml" target="_blank" rel="noopener"><code>mise.toml</code></a>
and installed by the bootstrap below.

## Why Tilt

- **A real programming language.** Tiltfiles are Starlark, not YAML: chart dependency builds, image
  builds and per-app live-update rules are shared functions instead of copy-pasted manifests.
- **Selective resources.** Core infra always runs; product apps and monitoring are opt-in, started
  by name or toggled from the Tilt UI.
- **Hot reload.** File changes sync into the running container without a rebuild.

## How it fits together

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/local-development.svg"
</div>

The root
<a href="https://github.com/kbntx-org/nexus/blob/main/Tiltfile" target="_blank" rel="noopener"><code>Tiltfile</code></a>
is a thin entrypoint; the real setup lives in
<a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/local" target="_blank" rel="noopener"><code>platform/core/local/</code></a>:
`platform.tilt` for core infra (its `CORE_RESOURCES`, `OPTIONAL_RESOURCES` and `PERSISTED_ON_DOWN`
lists are the source of truth), `apps.tilt` for product apps, and `lib.tilt` for the shared helpers.
An app with a more involved setup (smelt, for example) keeps its own Tiltfile, included from
`apps.tilt`.

## Getting started

**1. Bootstrap**

```bash
platform/core/local/local.sh create
```

It installs mise and every pinned tool, then creates the kind cluster, a local image registry and a
buildx builder, and trusts a local TLS certificate for `*.localhost`. It is idempotent: re-run it
(`mise run cluster create` once mise is installed) after a pinned version bump, or to re-trust the
certificate. Tilt's `watch-ingresses` resource reissues that certificate whenever a new hostname
appears, so a new app needs no re-bootstrap.

kind's own CNI and kube-proxy are disabled, so no pod schedules until Tilt installs Cilium; pods
stuck in `Pending` right after a bootstrap are expected.

**2. Start an app**

```bash
mise run dev docs
```

This is `tilt up -- docs`: core infra plus the named app. `tilt up` alone starts only core infra;
apps and monitoring can also be enabled later from the Tilt UI.

**3. Tear down**

`tilt down` removes everything except the resources in `PERSISTED_ON_DOWN`:

- **Cilium**: removing it would leave the cluster without pod networking.
- **CloudNative-PG**: a CNPG `Cluster` deletes its PVCs when it is deleted, and unlike a
  `StatefulSet` that can't be turned off. An app chart whose local database should survive
  `tilt down` annotates its `Cluster` with `helm.sh/resource-policy: keep` (smelt's does), and the
  operator has to stay too: uninstalling it would take the kept `Cluster`, and its volumes, down
  with it.

To start from scratch, `mise run cluster recreate` rebuilds the cluster, registry and builder;
`mise run cluster delete` removes them.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/Tiltfile" target="_blank" rel="noopener"><code>Tiltfile</code></a>
  — local dev entrypoint (resource enablement and what `tilt down` keeps)
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/local" target="_blank" rel="noopener"><code>platform/core/local/</code></a>
  — `lib.tilt`, `platform.tilt`, `apps.tilt`, and `local.sh`
- <a href="https://github.com/kbntx-org/nexus/blob/main/mise.toml" target="_blank" rel="noopener"><code>mise.toml</code></a>
  — every pinned local tool version, plus the `cluster` and `dev` tasks
