---
title: Container registry design
---

<a href="https://zotregistry.dev/" target="_blank" rel="noopener">zot</a> is our self-hosted OCI
registry. It holds every image this repository builds and caches the Docker Hub and GHCR images the
cluster pulls.

## How it's set up

zot runs as one pod with its storage in an R2 bucket, so the pod holds no image data. Its whole
configuration is one `config.json` in `values.yaml`, see zot's
<a href="https://zotregistry.dev/latest/admin-guide/admin-configuration/" target="_blank" rel="noopener">configuration
reference</a>.

- **Pushes:** the build job connects to WARP with a service token, logs in as `ci` and pushes.
- **Pulls:** the k3s role gives every node the read-only `node` credentials and mirrors `docker.io`
  and `ghcr.io` to zot's `/dockerhub` and `/ghcr` paths. The first pull of an upstream image caches
  it; later pulls are served from the cache, re-checked upstream at most hourly.

## Decisions and gotchas

- **zot over Harbor**: one binary with a UI, search and garbage collection, instead of a Postgres,
  Redis and Trivy stack to run.
- **Reachable over WARP only.** Through the public tunnel, Cloudflare would reject layers over 100
  MB, each sent as a single request.
- **Every request needs credentials** (htpasswd, no anonymous policy); only the health probes skip
  auth.
- **Retention:** the latest three tags of each built image, and cached upstream tags pulled within
  48 hours. A new built image must be added to the first policy by name, or zot keeps all its tags.
  With no pull statistics (after a restart) zot deletes nothing.
- **containerd falls back to the upstream registry when zot is down**, so the components zot depends
  on can still start. The cache pulls anonymously, so both paths are rate limited.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/zot/values.yaml" target="_blank" rel="noopener"><code>platform/core/zot/values.yaml</code></a>
  — `config.json`: storage, auth, retention, sync
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/kubernetes/configuration/roles/k3s/templates" target="_blank" rel="noopener"><code>roles/k3s/templates/</code></a>
  — `registries.yaml` mirrors on the nodes
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/warp/ci-access.tf" target="_blank" rel="noopener"><code>platform/core/warp/ci-access.tf</code></a>
  — CI's WARP service token
