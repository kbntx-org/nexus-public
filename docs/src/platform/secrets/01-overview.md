---
title: Overview
---

Secrets in Nexus follow one rule: **the cluster never owns the source of truth.** Values live in
<a href="https://developer.hashicorp.com/vault" target="_blank" rel="noopener">HashiCorp Vault</a>;
Git only declares _which_ secret a workload needs, and the
<a href="https://external-secrets.io/" target="_blank" rel="noopener">External Secrets Operator</a>
(ESO) materializes it at runtime. A namespace can be torn down and rebuilt without losing a
credential: it just re-reads it from Vault.

## Vault in the cluster

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/secrets.svg"
</div>

Vault runs in-cluster from the
<a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/vault/server" target="_blank" rel="noopener"><code>platform/core/vault/server/</code></a>
chart: the Vault Deployment, an `IngressRoute` for the API, and a
<a href="https://cloudnative-pg.io/" target="_blank" rel="noopener">CloudNativePG</a> `Cluster` for
storage.

**Why Postgres, not the embedded/Raft backends:** the platform already backs up Postgres, so reusing
it beats a Vault-specific snapshot workflow, and a remote backend keeps no data on a local disk. The
schema is created once by CNPG's `postInitApplicationSQLRefs` from
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/vault/server/files/vault.sql" target="_blank" rel="noopener"><code>vault.sql</code></a>,
skipped on recovered or re-attached volumes.

**The bootstrap secret.** Vault needs storage before it can serve its first request, so one secret
can't come from Vault: `vault-secret` in the `vault` namespace, applied manually. It holds the
Postgres connection URI, the CNPG superuser credentials and the R2 backup keys.

**Manual unseal.** Vault starts sealed on every boot. There is no auto-unseal: the deliberate
trade-off for keeping the unseal keys off the cluster entirely.

## External Secrets Operator

ESO reconciles two CRDs: a `SecretStore`/`ClusterSecretStore` (how to reach Vault) and an
`ExternalSecret` (what to fetch, and which Kubernetes `Secret` to build). Workloads mount that
`Secret` like any other, never knowing Vault is involved.

**Templating is the strong point.** An `ExternalSecret`'s `target.template` can inject a Vault value
into a larger document committed in Git, not just build a flat key/value `Secret`. Karpenter uses it
to render a whole cloud-init `userData` around a Vault-sourced k3s agent join token (see
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/karpenter/templates/secrets.yaml" target="_blank" rel="noopener"><code>karpenter/templates/secrets.yaml</code></a>).

### How ESO authenticates to Vault

Each consuming app has its own `ServiceAccount`. ESO presents its projected token (audience `vault`)
to Vault's
<a href="https://developer.hashicorp.com/vault/docs/auth/kubernetes" target="_blank" rel="noopener">Kubernetes
auth method</a>, and Vault validates it through the cluster's TokenReview API. That is why
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/external-secrets/deploy/templates/rbac.yaml" target="_blank" rel="noopener"><code>external-secrets/deploy/templates/rbac.yaml</code></a>
binds `system:auth-delegator` to the `external-secrets` ServiceAccount.

A single `eso` role accepts any `ServiceAccount`. Its templated policy grants read access only to
`platform/<namespace>` and `platform/<namespace>/*` in the KV, where `<namespace>` is the
authenticating `ServiceAccount`'s own namespace, so apps are isolated from each other without a role
per app. This Vault-side setup is long-lived and configured through Vault's own API, so Terraform
applies it once from
<a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/external-secrets/provision" target="_blank" rel="noopener"><code>platform/core/external-secrets/provision/</code></a>.

## Adding a new secret

Write the value into Vault under the KV path named after the consumer's namespace, then copy this
shape into the consumer's chart:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-app-secret-sa
automountServiceAccountToken: false
---
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: my-app-secret-store
spec:
  provider:
    vault:
      server: https://vault.example.com
      path: platform
      version: v2
      auth:
        kubernetes:
          mountPath: kubernetes
          role: eso
          serviceAccountRef:
            name: my-app-secret-sa
            audiences: [vault]
---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-app-secret
spec:
  refreshInterval: 30s
  secretStoreRef:
    name: my-app-secret-store
    kind: SecretStore
  target:
    name: my-app-secret
  dataFrom:
    - extract:
        key: my-app # matches the ServiceAccount's namespace
```

Real charts point `server` at the Vault `IngressRoute`'s hostname, set in
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/vault/server/values.yaml" target="_blank" rel="noopener">Vault's
values file</a>. The workload then consumes `my-app-secret`, and ESO refreshes it on its interval.
There is no Nexus-specific wrapper: the
<a href="https://external-secrets.io/latest/" target="_blank" rel="noopener">ESO docs</a> are
authoritative for templating, extraction and refresh behavior.

## Backups and disaster recovery

Vault's durability rests on its CNPG cluster, backed up like every other
([Databases](../databases/01-overview.md#backups)), and on the unseal keys and root token kept
out-of-band. To restore it, follow
[Restore a CNPG backup](../../runbooks/01-restore-a-cnpg-backup.md), then:

- **Unseal it by hand**: the Vault pod restarts sealed.
- **Re-apply `vault-secret`** if its `databaseUri` no longer matches: a side-by-side cut-over
  changes the host, and a rebuilt namespace loses the secret.
- **Write again anything stored after the restore point**: it is gone, and ESO picks the new values
  up on its next refresh.

Restore Vault before any other database: every other consumer reads its backup credentials through
ESO, which needs a working Vault. The `vault.sql` schema bootstrap only runs on a fresh cluster, so
a recovered one keeps the schema from the backup.

## Encryption at rest

k3s runs with `secrets-encryption: true`
(<a href="https://github.com/kbntx-org/nexus/blob/main/platform/modules/k3s/ansible/roles/k3s/templates/k3s-config.yml.j2" target="_blank" rel="noopener">k3s
config template</a>), so every Kubernetes `Secret`, including those ESO writes, is encrypted in etcd
rather than stored as plain base64. It is cluster-wide, not opt-in, and does **not** cover Postgres
data (see [Databases](../databases/01-overview.md#encryption-at-rest)).

## Local environment

Locally, Vault drops the CNPG cluster and runs in `-dev` mode on an in-memory backend, seeded by
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/vault/server/templates/seed-script.yaml" target="_blank" rel="noopener"><code>seed-script.yaml</code></a>,
so a fresh local cluster has a working Vault and ESO with zero operator steps.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/vault/server" target="_blank" rel="noopener"><code>platform/core/vault/server/</code></a>
  — Vault Helm chart — CNPG `Cluster` + `ScheduledBackup` for Vault's storage
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/external-secrets/deploy" target="_blank" rel="noopener"><code>platform/core/external-secrets/deploy/</code></a>
  — ESO Helm chart and TokenReview RBAC
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/external-secrets/provision/main.tf" target="_blank" rel="noopener"><code>platform/core/external-secrets/provision/main.tf</code></a>
  — the `eso` role and templated policy
