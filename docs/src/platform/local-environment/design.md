---
title: Local environment design
---

The local environment is a real Kubernetes cluster
(<a href="https://kind.sigs.k8s.io/" target="_blank" rel="noopener">kind</a>) driven by
<a href="https://tilt.dev/" target="_blank" rel="noopener">Tilt</a>, running the same ingress
controller and Helm charts as production. A `docker compose` setup was rejected because it drifts
from production; here a change is tested on the same pieces before it ships.

## How it's set up

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/local-development.svg"
</div>

Everything runs inside a
<a href="https://github.com/abiosoft/colima" target="_blank" rel="noopener">Colima</a> VM: the
checkout, Docker, kind, Tilt and any AI agent. Only the browser and the editor (VS Code Remote -
SSH) stay on the host. The root `Tiltfile` is a thin entrypoint; the setup lives in
`platform/core/local/`: `platform.tilt` for core infra, `apps.tilt` for apps, `mcp.tilt` for the MCP
servers agents use and `lib.tilt` for shared helpers. Core infra always runs; apps, monitoring and
the MCP servers are opt-in.

## Decisions and gotchas

- **The VM is the sandbox.** It mounts nothing from the host, so an agent can run any command
  without reaching the host's files, keychain or trust store, and a broken toolchain is fixed by
  recreating the VM. Running on a laptop with Docker still works, with nothing between an agent and
  the machine.
- **Tilt over plain manifests**: Tiltfiles are Starlark, so image builds, chart dependencies and
  live-update rules are shared functions, and file changes sync into running containers without a
  rebuild.
- **The certificate authority's key never leaves the VM.** The bootstrap issues a `*.localhost`
  certificate from a local
  <a href="https://github.com/FiloSottile/mkcert" target="_blank" rel="noopener">mkcert</a> CA; the
  host trusts only its public certificate, so nothing on the host can sign a new one. Tilt's
  `watch-ingresses` resource reissues the certificate when a new hostname appears.
- **`tilt down` keeps CloudNativePG.** A CNPG `Cluster` deletes its volumes with it, so a chart
  whose local database must survive annotates it `helm.sh/resource-policy: keep`, and the operator
  stays up (`PERSISTED_ON_DOWN`), or it would take the kept `Cluster` down with it.
- **The Playwright MCP server runs on the host network, not in the cluster.** Chromium resolves
  `*.localhost` to its own loopback, which in a pod is the pod. It ignores certificate errors: check
  TLS in your own browser.
- **The Grafana MCP server is one shared HTTP container per profile** (`tools/grafana-mcp.sh`), not
  one per session; it runs on your machine, so it reaches production over WARP and the token stays
  local.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/local" target="_blank" rel="noopener"><code>platform/core/local/</code></a>
  — Tilt files, `local.sh` and the Colima VM definition
- <a href="https://github.com/kbntx-org/nexus/blob/main/Tiltfile" target="_blank" rel="noopener"><code>Tiltfile</code></a>
  — entrypoint
- <a href="https://github.com/kbntx-org/nexus/blob/main/.mcp.json" target="_blank" rel="noopener"><code>.mcp.json</code></a>
  — MCP servers Claude Code connects to
