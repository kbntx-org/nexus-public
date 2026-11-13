---
title: Container registry
---

<a href="https://zotregistry.dev/" target="_blank" rel="noopener">zot</a> is the platform's
self-hosted OCI registry — a single static binary with no external database, exposing a browsable UI
and image search on top of the plain registry API. It runs as a
[core platform component](../cluster/01-overview.md#core-components), deployed the same way as every
other piece of cluster infrastructure (see [Overview](01-overview.md)).

## Why zot over an external registry

Docker Hub works, but a self-hosted registry gives the platform a place to hold artifacts that
shouldn't leave the cluster network (or shouldn't depend on a third party's uptime): cached upstream
images, internal build outputs, and anything the GitOps loop needs to pull from a source it fully
controls. Every image this repo builds lives only in zot, and Docker Hub is only an upstream the
cache reads from. zot was picked over heavier options like
<a href="https://goharbor.io/" target="_blank" rel="noopener">Harbor</a> because it ships the same
core needs — a UI, browsing, garbage collection, and optional auth — without a multi-service
Postgres + Redis + Trivy stack to operate.

## Configuration

The chart mounts a single `config.json` (see zot's
<a href="https://zotregistry.dev/latest/admin-guide/admin-configuration/" target="_blank" rel="noopener">configuration
reference</a>) rather than exposing every option as a Helm value — that file is the source of truth
for storage and the extensions below.

- **Authentication** — `http.auth.htpasswd` reads a bcrypt htpasswd file mounted at
  `/secret/htpasswd`, and `http.accessControl` has no anonymous policy, so every request needs
  credentials, which matters because the registry is reachable from the internet (see
  [Access](#access)). Two users exist: `ci` can read, create, update and delete, and `node` (used by
  the cluster nodes to pull) can only read. The file is built by the `zot-htpasswd` `ExternalSecret`
  from the two hashes in Vault. The health endpoints the probes use (`/livez`, `/readyz`,
  `/startupz`) are served before the auth middleware, so the probes need no credentials. zot also
  supports OIDC and LDAP, see the
  <a href="https://zotregistry.dev/latest/articles/authn-authz/" target="_blank" rel="noopener">authn/authz
  guide</a>.
- **Garbage collection** — `storage.gc` reclaims blobs orphaned by manifest deletion on a schedule
  (`gcInterval`), with `gcDelay` giving in-flight pulls time to finish before a blob is swept.
- **UI and search** — the `extensions.ui` and `extensions.search` blocks turn on the registry's
  browsable web UI, which depends on the search extension being enabled to index repositories.
- **Storage** — `storage.storageDriver` points zot at an R2 bucket, the same object storage used for
  database backups (see [Databases](../databases/01-overview.md)), instead of a
  `PersistentVolumeClaim`, so the pod holds no image data itself. The chart still mounts an
  `emptyDir` at `storage.rootDirectory` for zot's local blob-metadata cache, which is safe to lose
  on a restart — the registry content itself lives entirely in the bucket. Credentials for that
  bucket come in as `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` env vars from a Vault-backed secret
  (see [Secrets management](../secrets/01-overview.md)) rather than living in `config.json` itself.
- **Pull-through cache** — `extensions.sync` mirrors Docker Hub and GHCR on demand: the first pull
  of an upstream image through zot fetches and caches it, every pull after that is served locally. A
  cached tag is re-checked against its upstream at most once an hour (`manifestCheckInterval`), so a
  pull no longer contacts Docker Hub or GHCR each time. Each upstream is routed to its own path
  (`/dockerhub`, `/ghcr`) via `content[].destination`, so `registry.kbntx.com/dockerhub/<image>` and
  `registry.kbntx.com/ghcr/<image>` behave like local mirrors of their upstream. See zot's
  <a href="https://zotregistry.dev/latest/articles/mirroring/" target="_blank" rel="noopener">mirroring
  guide</a> for the full `sync` schema (scheduled sync, tag filters, credentials for private
  upstreams).

## Access

zot is private, like the other internal apps: its hostname resolves to the internal ingress and is
only reachable over [WARP](../traffic/01-overview.md#private-access-via-warp). Exposing it through
the public tunnel instead would put every push behind Cloudflare's proxy, which rejects request
bodies over 100 MB on the Free and Pro plans, and a layer is sent as a single request.

- **Pushes** — each project's `build-ci` command tags its image with the bare image name in zot. The
  build workflow connects the runner to WARP with a service token
  (`platform/core/warp/ci-access.tf`), then logs in as `ci` with the `ZOT_CI_USER` and
  `ZOT_CI_PASSWORD` repository secrets. Pull requests connect and log in too, but don't push.
- **Pulls** — the k3s role writes the `node` credentials into `registries.yaml` when
  `k3s_zot_password` is set (see
  [Run the cluster Ansible playbook](../../runbooks/02-run-the-cluster-ansible-playbook.md)), and
  mirrors `docker.io` and `ghcr.io` to the cache, so every Docker Hub and GHCR image a pod asks for
  is pulled through zot. containerd always tries the original registry last, so a pull still
  succeeds when zot is down, which keeps the platform components zot itself depends on from
  deadlocking. The cache has no upstream credentials, so the fetches it makes are anonymous and rate
  limited; the nodes' own Docker Hub credentials only apply to that fallback.

## Provisioning the R2 bucket

Unlike every other chart's `templates/secrets.yaml`, which only reads Vault, zot also needs a
Cloudflare R2 bucket (`nexus-registry`) and a bucket-scoped S3 credential (Object Read & Write) to
exist before it can start. They are created by hand in Cloudflare, outside the GitOps loop.

Everything is stored in the `platform` KV mount under the `zot` key:

| Field               | Value                                                                   |
| ------------------- | ----------------------------------------------------------------------- |
| `s3AccessKeyId`     | Access key ID of the R2 token                                           |
| `s3SecretAccessKey` | Secret access key of the R2 token                                       |
| `ciPasswordHash`    | bcrypt hash of the `ci` user's password (`htpasswd -nbB ci <password>`) |
| `nodePasswordHash`  | bcrypt hash of the `node` user's password                               |

Store only the hash part of the `htpasswd` output (everything after `user:`). The plain passwords
belong in a password manager, and the `ci` one also in the CI secrets.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/zot" target="_blank" rel="noopener"><code>platform/core/zot/</code></a>
  — the umbrella chart, Vault-backed secrets, and ingress
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/zot/values.yaml" target="_blank" rel="noopener"><code>platform/core/zot/values.yaml</code></a>
  — `config.json`, storage backend, sync, and ingress host
- [Overview](01-overview.md) — the app-of-apps pattern this component is registered under
- [Secrets management](../secrets/01-overview.md) — the Vault + External Secrets pattern reused here
