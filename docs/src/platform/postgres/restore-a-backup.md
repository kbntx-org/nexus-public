---
title: Restore a backup
---

Bring a Postgres cluster back from R2, to the latest state or a point in time. CNPG never restores
in place: recovery always bootstraps a new `Cluster` from a base backup and the archived WAL. Pick
one:

- **Side by side**: a temporary `<cluster>-restore` next to the original, to inspect or copy data
  back. Nothing is destroyed: start here when in doubt.
- **Replace**: delete the original and recreate it, same name, from the backups. Use it when the
  database is lost or corrupt.

Restoring Vault? Do it before any other database, then see
[Restore Vault](../vault/how-to.md#restore-vault-platform).

## Before you start

- WARP and `kubectl` access, and write access to `main`: ArgoCD's `selfHeal` reverts live changes,
  so everything goes through Git except pausing auto-sync.
- From the consumer's chart: the `Cluster` name and database (`postgres-cnpg.yaml`),
  `backup.destinationPath` and `backup.endpointURL` (`values.yaml`).
- For a point in time: an RFC 3339 timestamp with a timezone (`2026-01-31T08:00:00Z`), within the
  retention window.

## Side-by-side restore

1. Add `templates/postgres-cnpg-restore.yaml` next to the original, filled from it:

   ```yaml
   apiVersion: postgresql.cnpg.io/v1
   kind: Cluster
   metadata:
     name: <cluster>-restore
   spec:
     instances: 1
     imageName: <same image as the original>
     storage:
       size: <same size>
       storageClass: <same storage class>
     bootstrap:
       recovery:
         source: origin
         database: <database>
         owner: <database>
         recoveryTarget: # omit to replay to the end of the archive
           targetTime: '<YYYY-MM-DDTHH:MM:SSZ>'
     externalClusters:
       - name: origin
         barmanObjectStore:
           destinationPath: <backup.destinationPath>
           endpointURL: <backup.endpointURL>
           serverName: <cluster>
           s3Credentials:
             accessKeyId:
               name: <backup credentials secret>
               key: backupAccessKeyId
             secretAccessKey:
               name: <backup credentials secret>
               key: backupSecretAccessKey
           wal:
             compression: gzip
   ```

   It has **no `backup` block**, so it never archives into the original's path.

2. Push to `main`. Recovery runs in a `<cluster>-restore-1-full-recovery` pod.
3. [Verify](#verify), then `pg_dump` what you need from `<cluster>-restore-1` and load it into the
   original with `psql`. For a full rollback, use [Replace](#replace-the-cluster) instead of
   pointing the consumer at the `-restore` cluster.
4. Delete the template and push. ArgoCD prunes the cluster and its volume.

## Replace the cluster

**This destroys the current database.** Run a side-by-side restore first if you're unsure of the
backups.

1. **Pause auto-sync** of the consumer's app in the ArgoCD UI.
2. If the original still runs, archive its last WAL and wait for `completed`:

   ```sh
   kubectl apply -n <namespace> -f - <<'EOF'
   apiVersion: postgresql.cnpg.io/v1
   kind: Backup
   metadata:
     name: <cluster>-before-restore
   spec:
     cluster:
       name: <cluster>
   EOF
   kubectl get backup -n <namespace> <cluster>-before-restore -w
   ```

3. On `main`, without syncing: replace `bootstrap.initdb` in `postgres-cnpg.yaml` with the
   `bootstrap.recovery` and `externalClusters` blocks above, reading the **current** path, and point
   `backup.destinationPath` at a **new** one (`s3://<bucket>/<consumer>-cnpg-<yyyymmdd>`).
4. Stop the consumer and delete the cluster:

   ```sh
   kubectl scale deployment -n <namespace> <consumer> --replicas=0
   kubectl delete cluster -n <namespace> <cluster>
   kubectl get pvc -n <namespace>   # delete any PVC of <cluster> left behind
   ```

5. Sync the app in the ArgoCD UI, then [verify](#verify).
6. Put `bootstrap.initdb` back (CNPG reads `bootstrap` only at creation), keep the new path, push
   and turn auto-sync back on.

!!! warning "Not yet exercised"

    The replace path hasn't been run end to end. Watch whether CNPG accepts switching `bootstrap`
    back in step 6 (if not, leave the recovery blocks until the next rebuild) and whether step 4
    removed the PVCs. Update this page with what you observe.

## Verify

```sh
kubectl get cluster -n <namespace> <cluster>
kubectl logs -n <namespace> -l cnpg.io/cluster=<cluster> --tail=100
kubectl exec -it -n <namespace> <cluster>-1 -- psql -U postgres -d <database> \
  -c 'SELECT count(*) FROM <a table you know>;'
```

The cluster is `Cluster in healthy state`, the data matches the restore point, and the consumer
works end to end. After a replace, `ContinuousArchiving` is `True` in the cluster's status.

## Gotchas

- **A recovered cluster must not archive into the original's folder.** CNPG stops at
  `Setting up primary` with `Expected empty archive`. Never set `cnpg.io/skipEmptyWalArchiveCheck`:
  two timelines would overwrite each other.
- **The old archive is left alone** after a replace; delete it from R2 when you no longer need it.
- **Set `database` and `owner`**, or CNPG also creates an empty `app` database and user.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/n8n/templates/postgres-cnpg.yaml" target="_blank" rel="noopener"><code>n8n/templates/postgres-cnpg.yaml</code></a>
  — a consumer to copy settings from

**External**:

- <a href="https://cloudnative-pg.io/docs/devel/recovery" target="_blank" rel="noopener">CNPG
  recovery</a>
