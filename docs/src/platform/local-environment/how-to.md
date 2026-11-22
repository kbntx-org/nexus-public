---
title: Local environment
---

Run the platform and any app on your machine in a kind cluster, inside a Colima VM. Steps marked
**host** run on your machine; everything else runs in the VM, from a VS Code terminal connected to
it.

## Set up

1. **Create the VM (host).** Install Colima, Lima, the Docker CLI and VS Code Remote - SSH. Colima
   reads its configuration from its profile directory, so copy
   <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/local/colima.yaml" target="_blank" rel="noopener"><code>colima.yaml</code></a>
   there (again, followed by `colima restart`, whenever it changes):

   ```bash
   mkdir -p ~/.colima/default
   cp platform/core/local/colima.yaml ~/.colima/default/colima.yaml
   colima start
   ```

   It targets Apple silicon; lower CPU, memory and disk on a smaller machine. In VS Code, connect to
   the `colima` SSH host and clone the repository inside the VM.

2. **Bootstrap (VM).** `platform/core/local/local.sh create` installs mise and every pinned tool,
   then creates the kind cluster, a local registry, a Buildx builder and the `*.localhost`
   certificate. It is idempotent: re-run it after a version bump.

3. **Trust the certificate authority (host)**, with mkcert installed on the host. Firefox also needs
   `nss`.

   ```bash
   mkdir -p /tmp/nexus-mkcert
   colima ssh -- sh -c 'cat "$HOME/.local/share/mkcert/rootCA.pem"' > /tmp/nexus-mkcert/rootCA.pem
   CAROOT=/tmp/nexus-mkcert mkcert -install
   ```

   mkcert warns that the key is missing, which is expected. Do it once per VM: a new VM generates a
   new CA, `local.sh recreate` keeps it.

## Start an app

```bash
tilt up -- docs
```

Core infra plus the named resources, here the docs at `https://docs.localhost` (Colima forwards
ports 80 and 443 to the host). Name more to start them too (`tilt up -- docs monitoring`), or toggle
them in the Tilt UI.

## Add an app to Tilt

In `platform/core/local/apps.tilt`, add the app to `PRODUCT_RESOURCES`, then a `buildkit_build` for
its image (with `live_update` sync for hot reload) and a `helm_resource` for its chart with its
`values.local.yaml`; copy the `kiln` block. An app with a more involved setup keeps its own
Tiltfile, included from `apps.tilt`.

## Start over

`tilt down` removes everything except [CloudNativePG](design.md#decisions-and-gotchas).
`platform/core/local/local.sh recreate` rebuilds the cluster, registry and builder;
`local.sh delete` removes them.

## Let an agent test in a browser

Start the `playwright-mcp` resource with the app (`tilt up -- docs playwright-mcp`). Claude Code
connects to it through `.mcp.json` and can click through a flow, read network requests and the
console and take screenshots, saved in the gitignored `.playwright-mcp/` directory.

## Let an agent query Grafana

Start `grafana-mcp-local` (with `monitoring`) or `grafana-mcp-prod`. Both read their settings from
the gitignored `env.yaml` at the repository root, falling back to
<a href="https://github.com/kbntx-org/nexus/blob/main/env.example.yaml" target="_blank" rel="noopener"><code>env.example.yaml</code></a>.
For production, with WARP on:

1. In the production Grafana, create a service account with the **Viewer** role (**Administration →
   Users and access → Service accounts**) and add a token. Viewer keeps the agent read-only.
2. Copy `env.example.yaml` to `env.yaml` and set `grafana.prod.url` and
   `grafana.prod.serviceAccountToken`.

## Run an isolated environment

An autonomous task on its own worktree gets its own Docker daemon, kind cluster and Tilt in a
`docker run --runtime=sysbox-runc` container: inside it, `local.sh create` then `tilt up`. Never use
`--privileged` or mount the Docker socket: both give the container control of the VM.

How it works: [Local environment design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/mise.toml" target="_blank" rel="noopener"><code>mise.toml</code></a>
  — pinned tools
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/local/local.sh" target="_blank" rel="noopener"><code>platform/core/local/local.sh</code></a>
  — cluster, registry, builder and certificate bootstrap
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/local/apps.tilt" target="_blank" rel="noopener"><code>platform/core/local/apps.tilt</code></a>
  — the apps Tilt can start
