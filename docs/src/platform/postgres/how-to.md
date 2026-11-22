---
title: Postgres
---

Each app that needs Postgres gets its own CloudNativePG cluster, backed up daily to R2. To get data
back, see [Restore a backup](restore-a-backup.md).

## Add a database to an app

1. Copy the `Cluster` and `ScheduledBackup` in
   <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/n8n/templates/postgres-cnpg.yaml" target="_blank" rel="noopener"><code>n8n/templates/postgres-cnpg.yaml</code></a>
   and change the cluster name, database, superuser secret and storage size; set
   `postgresCluster.storageClass` in the values.
2. Add `backup.enabled`, `backup.destinationPath` (a new path in the backup bucket) and
   `backup.endpointURL` to `values.yaml`, with `backup.enabled: false` in `values.local.yaml`.
3. Put the bucket credentials (`backupAccessKeyId`, `backupSecretAccessKey`) and the superuser's
   `postgresUser` and `postgresPassword` in the app's Vault path, and read them with
   `ExternalSecret`s, the superuser one as a `basic-auth` Secret as n8n does (see
   [Vault](../vault/how-to.md)).
4. Add the database's network policy: copy the `n8n-postgres-cnpg` one in
   `n8n/templates/network-policy.yaml` (the app and `cloudflared` on 5432, the operator on 8000, the
   API server on 6443, R2 on 443 for backups), and let the app reach it on 5432.
5. Point the app at `<cluster>-rw` with the credentials from its secret, as n8n does with its
   superuser secret. CNPG also generates a `<cluster>-app` Secret for the database owner.
6. To keep the local database across `tilt down`, annotate the `Cluster`
   [`helm.sh/resource-policy: keep`](../local-environment/design.md#decisions-and-gotchas).

## Connect to a database

Over WARP, the `<cluster>-rw` Service is reachable directly. For a shell:

```sh
kubectl exec -it -n <namespace> <cluster>-1 -- psql -U postgres -d <database>
```

How it works: [Postgres design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/n8n/values.yaml" target="_blank" rel="noopener"><code>n8n/values.yaml</code></a>
  — a consumer's `backup` values

**External**:

- <a href="https://cloudnative-pg.io/documentation/current/" target="_blank" rel="noopener">CloudNativePG
  documentation</a>
