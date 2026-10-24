---
title: Local Development
---

The local environment runs inside a real Kubernetes cluster
(<a href="https://kind.sigs.k8s.io/" target="_blank" rel="noopener">kind</a>), orchestrated by
<a href="https://tilt.dev/" target="_blank" rel="noopener">Tilt</a>, with the **same ingress
controller and Helm charts** as production. A parallel `docker compose` setup was rejected because
it drifts from the real thing; here a change is tested against the same primitives before it ships.

The supported setup runs everything inside a
<a href="https://github.com/abiosoft/colima" target="_blank" rel="noopener">Colima</a> VM: the
repository, Docker, the kind cluster, Tilt and any AI agent working on the code. Only the browser
and the editor stay on the host. The VM is the sandbox: an agent can run commands freely without
reaching the host's processes, files, keychain or trust store (the VM mounts nothing from the host),
and a broken toolchain is fixed by recreating the VM rather than cleaning up the host. Running
directly on a laptop with Docker still works: skip the steps marked **host** below, but an agent
then works with nothing between it and the rest of the machine.

The VM's prerequisites on the host are Colima,
<a href="https://lima-vm.io/" target="_blank" rel="noopener">Lima</a>, the Docker CLI and
<a href="https://code.visualstudio.com/docs/remote/ssh" target="_blank" rel="noopener">VS Code
Remote - SSH</a> to develop inside the VM. Colima and Lima are pinned in
<a href="https://github.com/kbntx-org/nexus/blob/main/mise.toml" target="_blank" rel="noopener"><code>mise.toml</code></a>
with every other tool, which the bootstrap below installs in the VM.

### Isolated environments in the VM

The VM runs the
<a href="https://github.com/nestybox/sysbox" target="_blank" rel="noopener">Sysbox</a> container
runtime, so an autonomous task (an agent working on its own git worktree, for example) gets a whole
environment of its own: a `docker run --runtime=sysbox-runc` container runs its own Docker daemon,
kind cluster and Tilt without touching the VM's cluster or clashing with its ports. The usual
alternatives, a `--privileged` container or one with the Docker socket mounted, are forbidden: both
hand the container control of the VM.

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

Steps marked **host** run on your machine; every other command runs in the VM, from a VS Code
terminal connected to it.

**1. Create the VM and connect to it (host)**

The VM is defined by
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/local/colima.yaml" target="_blank" rel="noopener"><code>platform/core/local/colima.yaml</code></a>:
its size, no host mounts, and a boot script that installs Sysbox. Colima reads its configuration
from its profile directory, so copy the file there before the first start:

```bash
mkdir -p ~/.colima/default
cp platform/core/local/colima.yaml ~/.colima/default/colima.yaml
colima start
```

The file targets Apple silicon (the `vz` VM type with Rosetta); lower its CPU, memory and disk on a
smaller machine. `colima start` adds a `colima` host to `~/.ssh/config`: in VS Code, open the Remote
Explorer, connect to it and clone the repository inside the VM (`~/workspace/nexus`, for example).

**2. Bootstrap (VM)**

```bash
platform/core/local/local.sh create
```

It installs mise and every pinned tool, then creates the kind cluster, a local image registry and a
buildx builder, and issues a TLS certificate for `*.localhost` signed by a local
<a href="https://github.com/FiloSottile/mkcert" target="_blank" rel="noopener">mkcert</a>
certificate authority. It is idempotent: re-run it (`mise run cluster create` once mise is
installed) after a pinned version bump. Tilt's `watch-ingresses` resource reissues that certificate
whenever a new hostname appears, so a new app needs no re-bootstrap.

kind's own CNI and kube-proxy are disabled, so no pod schedules until Tilt installs Cilium; pods
stuck in `Pending` right after a bootstrap are expected.

**3. Trust the certificate authority (host)**

The bootstrap's `mkcert -install` trusts the CA inside the VM only, but the browser runs on the
host. Trust the CA's public certificate there, with mkcert installed on the host
(`brew install mkcert`):

```bash
mkdir -p /tmp/nexus-mkcert
colima ssh -- sh -c 'cat "$HOME/.local/share/mkcert/rootCA.pem"' > /tmp/nexus-mkcert/rootCA.pem
CAROOT=/tmp/nexus-mkcert mkcert -install
```

Only `rootCA.pem` leaves the VM; the CA's private key stays there, so certificates are still issued
in the VM and nothing on the host can sign new ones. mkcert warns that the key is missing, which is
expected. This is needed once per CA: `mise run cluster recreate` keeps it, but a new VM generates a
new one. Firefox uses its own trust store and needs `nss` installed (`brew install nss`) before
running `mkcert -install`.

Colima forwards the ports the VM listens on to the host's `localhost`, so the kind node's ports 80
and 443 are reachable from the host browser at `https://*.localhost` with no extra setup.

**4. Start an app (VM)**

```bash
mise run dev docs
```

This is `tilt up -- docs`: core infra plus the named app. `tilt up` alone starts only core infra;
apps and monitoring can also be enabled later from the Tilt UI.

**5. Tear down (VM)**

`tilt down` removes everything except the resources in `PERSISTED_ON_DOWN`:

- **Cilium**: removing it would leave the cluster without pod networking.
- **CloudNative-PG**: a CNPG `Cluster` deletes its PVCs when it is deleted, and unlike a
  `StatefulSet` that can't be turned off. An app chart whose local database should survive
  `tilt down` annotates its `Cluster` with `helm.sh/resource-policy: keep` (smelt's does), and the
  operator has to stay too: uninstalling it would take the kept `Cluster`, and its volumes, down
  with it.

To start from scratch, `mise run cluster recreate` rebuilds the cluster, registry and builder;
`mise run cluster delete` removes them.

## Browser for AI agents

The optional `playwright-mcp` resource runs the
<a href="https://github.com/microsoft/playwright-mcp" target="_blank" rel="noopener">Playwright MCP
server</a>, a headless Chromium an AI agent drives to smoke test the app it changed: it clicks
through a flow, fills in forms, inspects network requests and the console, and takes screenshots at
any viewport, so a change is exercised end to end, not just compiled and unit tested. Start it with
the app (`mise run dev docs playwright-mcp`) or from the Tilt UI/CLI; the repository's
<a href="https://github.com/kbntx-org/nexus/blob/main/.mcp.json" target="_blank" rel="noopener"><code>.mcp.json</code></a>
connects Claude Code to it.

It runs as a plain Docker container on the host network rather than in the cluster: Chromium always
resolves `*.localhost` to its own loopback, which in a pod is the pod itself, while on the host
network it is where kind publishes Traefik. The image has no way to trust the mkcert certificate
authority, so certificate errors are ignored; check a TLS problem from your own browser. Page
snapshots, console logs and screenshots land in the gitignored `.playwright-mcp/` directory, where
the agent reads them.

## Grafana for AI agents

The optional `grafana-mcp` resource runs the
<a href="https://github.com/grafana/mcp-grafana" target="_blank" rel="noopener">Grafana MCP
server</a>, so an AI agent can search dashboards, run metric and log queries and read alert rules
against a Grafana instance, for instance to check that a change emits the metrics and logs it
should, or to investigate a production incident. Start it with `tilt enable grafana-mcp` or from the
Tilt UI; `.mcp.json` connects Claude Code to it.

Which Grafana it signs in to comes from the gitignored `env.yaml` at the repository root, falling
back to
<a href="https://github.com/kbntx-org/nexus/blob/main/env.example.yaml" target="_blank" rel="noopener"><code>env.example.yaml</code></a>,
which targets the local stack (start monitoring too) with its dev admin credentials. To target
production, with WARP on:

1. Sign in to the production Grafana with GitHub and create a service account with the **Viewer**
   role under **Administration → Users and access → Service accounts**, then add a token.
2. Copy `env.example.yaml` to `env.yaml`, set `grafana.url` to the production Grafana and replace
   the username and password with `grafana.serviceAccountToken`. Tilt restarts the server when it
   changes.

The Viewer role keeps the agent read-only. `env.yaml` holds every local setting of this kind,
validated by
<a href="https://github.com/kbntx-org/nexus/blob/main/.yaml-schemas/env.schema.json" target="_blank" rel="noopener"><code>.yaml-schemas/env.schema.json</code></a>,
which the example references for editor completion; a new setting goes in the schema and the example
together.

It runs as a Docker container on the host network rather than in the cluster, so it reaches the
internal production Grafana through WARP like your browser does, and the token stays on your
machine. The VM's CA bundle is mounted in, so it trusts the mkcert certificate locally and the
public one in production. The server has no SSO: anything on your machine that can reach its port
acts with that token.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/Tiltfile" target="_blank" rel="noopener"><code>Tiltfile</code></a>
  — local dev entrypoint (resource enablement and what `tilt down` keeps)
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/local" target="_blank" rel="noopener"><code>platform/core/local/</code></a>
  — `lib.tilt`, `platform.tilt`, `apps.tilt`, and `local.sh`
- <a href="https://github.com/kbntx-org/nexus/blob/main/mise.toml" target="_blank" rel="noopener"><code>mise.toml</code></a>
  — every pinned local tool version, plus the `cluster` and `dev` tasks
