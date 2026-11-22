---
title: Vault
---

Vault holds every secret; External Secrets copies each one into a Kubernetes Secret. The UI is
reachable over WARP.

## Add a secret

Write it in the `platform` KV mount under the key named after the app's namespace
(`platform/<namespace>`, or a sub-key). The app can read nothing else.

## Use a secret in a chart

Copy an existing `templates/secrets.yaml`, such as
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/n8n/templates/secrets.yaml" target="_blank" rel="noopener"><code>n8n/templates/secrets.yaml</code></a>:

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
      server: https://vault.example.com # the Vault route's host
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
        key: my-app # the namespace
```

A changed value reaches the Secret within `refreshInterval`; restart the pod if it reads the value
only at start.

Locally, Vault runs in dev mode with a `dev` mount and an `admin` role. Make the store's server,
path and role chart values, set them in `values.local.yaml`, and seed the local values in
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/vault/server/templates/seed-script.yaml" target="_blank" rel="noopener"><code>seed-script.yaml</code></a>.
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/monitoring/templates/secrets.yaml" target="_blank" rel="noopener"><code>monitoring/templates/secrets.yaml</code></a>
and its `values.local.yaml` show the pattern.

!!! warning "Escape ESO templates in Helm"

    ESO's `target.template` and Helm both use Go templates. Inside a chart, write
    ``{{ `{{ .key }}` }}``, or Helm renders it first and the Secret comes out empty.

## Debug a secret that doesn't sync

`kubectl describe externalsecret -n <namespace> <name>`: a `SecretSyncedError` usually means a wrong
key or a key outside the app's namespace. Login failures and a sealed Vault show on the store:
`kubectl describe secretstore -n <namespace>`.

## Unseal Vault (platform)

After every restart, with WARP on, enter the unseal keys in the UI or run `vault operator unseal`
until it reports unsealed. Keys and root token are kept out of band.

## Restore Vault (platform)

Restore its database first ([Restore a backup](../postgres/restore-a-backup.md)), then:

- unseal it;
- update `vault-secret` if its `databaseUri` changed (a side-by-side cut-over changes the host, a
  rebuilt namespace loses the secret);
- write again anything stored after the restore point.

How it works: [Vault design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/external-secrets/provision/main.tf" target="_blank" rel="noopener"><code>external-secrets/provision/main.tf</code></a>
  — the `eso` role and its policy

**External**:

- <a href="https://external-secrets.io/latest/" target="_blank" rel="noopener">External Secrets
  Operator</a>
