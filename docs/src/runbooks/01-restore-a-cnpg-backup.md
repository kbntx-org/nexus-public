---
title: Restore a CNPG backup
---

Bring a Postgres cluster back from its R2 backups, to the latest state or to a point in time. Every
cluster is single-instance, so this is the only recovery path (see
[Databases](../platform/databases/01-overview.md#backups) for how backups are taken).

CloudNativePG never restores in place: recovery always **bootstraps a new `Cluster`** from a base
backup plus the archived WAL
(<a href="https://cloudnative-pg.io/docs/devel/recovery" target="_blank" rel="noopener">CNPG
recovery docs</a>). Pick the path that matches the damage:

- **Side-by-side**: a temporary `<cluster>-restore` next to the original. Nothing is destroyed; use
  it to inspect old data or copy a few rows back. Start here when in doubt.
- **Replace**: delete the original and recreate it, same name, from the backups. The consumer keeps
  its connection settings. Use it when the database is lost or corrupt as a whole.

## Before you start

- `kubectl` access to the cluster over WARP (see
  [Traffic & Access](../platform/traffic/01-overview.md#private-access-via-warp)).
- Write access to `main`: ArgoCD's `selfHeal` reverts any live change to a GitOps-managed resource,
  so every change below goes through Git.
- The consumer's values: its `Cluster` name and database in its `postgres-cnpg.yaml`, and
  `backup.destinationPath` / `backup.endpointURL` in its `values.yaml`.
- The point to restore to, if not the latest: an RFC 3339 timestamp **with a timezone**
  (`2026-01-31T08:00:00Z`). It must be within the `retentionPolicy` window of the template.

## Side-by-side restore

1. Add a temporary template next to the consumer's `postgres-cnpg.yaml`, for example
   `templates/postgres-cnpg-restore.yaml`. Fill the placeholders from the original `Cluster` and the
   chart's values, taking the R2 credentials secret from its `backup` block:

   ```yaml
   apiVersion: postgresql.cnpg.io/v1
   kind: Cluster
   metadata:
     name: <cluster>-restore
   spec:
     instances: 1
     imageName: <same image as the original>
     storage:
       size: <same size as the original>
       storageClass: <same storage class as the original>
     bootstrap:
       recovery:
         source: origin
         database: <database>
         owner: <database>
         # Optional point in time; omit to replay to the end of the archive.
         recoveryTarget:
           targetTime: '<YYYY-MM-DDTHH:MM:SSZ>'
     externalClusters:
       - name: origin
         barmanObjectStore:
           destinationPath: <original backup.destinationPath>
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

   It has **no `backup` block on purpose**: it must never archive into the original's path (see
   [Gotchas](#gotchas)). `serverName` is the original cluster's name, the folder its backups live
   under.

2. Commit and push to `main`. ArgoCD creates the cluster; recovery runs in a
   `<cluster>-restore-1-full-recovery` pod.
3. [Verify](#verify) it, then use it:
   - **Copy data back:** `pg_dump` the tables you need from `<cluster>-restore-1` and load them into
     the original with `psql`.
   - **Cut over:** pointing the consumer at `<cluster>-restore-rw` means changing its connection
     host, which lives in a different place per consumer, and keeping the `-restore` name for good.
     Prefer [Replace](#replace-the-cluster) for a full rollback.
4. Delete the temporary template and push. ArgoCD prunes the `Cluster`, and CNPG deletes its PVC.

## Replace the cluster

**This destroys the current database.** If you are not sure the backups hold what you need, run a
side-by-side restore first.

1. **Pause auto-sync** for the consumer's `Application`: remove its `syncPolicy.automated` block in
   <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/app-of-apps/values.yaml" target="_blank" rel="noopener"><code>app-of-apps/values.yaml</code></a>
   and push. Check in the ArgoCD UI that the app shows auto-sync disabled before going on.
2. If the original still runs, archive its last WAL with an on-demand backup and wait for
   `completed`:

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

3. Change the chart on `main`, without syncing yet:
   - in `postgres-cnpg.yaml`, replace `bootstrap.initdb` with the `bootstrap.recovery` and
     `externalClusters` blocks of the side-by-side template above, still reading from the
     **current** path;
   - in `values.yaml`, point `backup.destinationPath` at a **new** path, for example
     `s3://<bucket>/<consumer>-cnpg-<yyyymmdd>`.
4. Stop the consumer and delete the cluster:

   ```sh
   kubectl scale deployment -n <namespace> <consumer> --replicas=0
   kubectl delete cluster -n <namespace> <cluster>
   kubectl get pvc -n <namespace>   # delete any PVC of <cluster> left behind
   ```

5. Sync the app from the ArgoCD UI. The cluster comes back from the backups under the same name and
   the deployment returns to its declared replicas. [Verify](#verify).
6. Put back `bootstrap.initdb` in place of the recovery blocks (CNPG reads `bootstrap` only at
   creation), keep the new `destinationPath`, restore the `automated` block in `app-of-apps`, and
   push.

!!! warning "Not yet exercised"

    The replace path has not been run end to end on this platform. Two points to watch the first
    time: whether CNPG accepts switching `bootstrap` back to `initdb` on the existing `Cluster` in
    step 6 (if it rejects it, leave the recovery blocks in place until the next planned rebuild),
    and whether deleting the `Cluster` in step 4 removed its PVCs. Update this page with what you
    observe.

## Verify

```sh
kubectl get cluster -n <namespace> <cluster>
kubectl logs -n <namespace> -l cnpg.io/cluster=<cluster> --tail=100
kubectl exec -it -n <namespace> <cluster>-1 -- psql -U postgres -d <database> \
  -c 'SELECT count(*) FROM <a table you know>;'
```

- The cluster reports `Cluster in healthy state`.
- The row count, or the latest rows, match what you expect at the restore point.
- The consumer works end to end: log in, open a page that reads the restored data.
- After a replace, the new archive fills: the `ContinuousArchiving` condition in
  `kubectl get cluster -n <namespace> <cluster> -o yaml` is `True`.

## Gotchas

- **A recovered cluster must not archive into the original's folder.** CNPG refuses a non-empty
  archive: the pod stops at `Setting up primary` with
  `WAL archive check failed for server <cluster>: Expected empty archive`. Hence no `backup` block
  on the side-by-side cluster and a new `destinationPath` on a replace. Never set the
  `cnpg.io/skipEmptyWalArchiveCheck` annotation: it lets two timelines overwrite each other.
- **The old archive is left alone.** After a replace, no cluster applies the retention policy to the
  previous path any more; delete it from R2 once you no longer need it.
- **Set `database` and `owner` in `bootstrap.recovery`.** They default to `app`, and CNPG would then
  create an empty `app` database and user next to the restored one.
- **This uses CNPG's in-tree `barmanObjectStore`**, like the backups. Upstream deprecated it in
  favor of the Barman Cloud plugin; recovery still follows the
  <a href="https://cloudnative-pg.io/docs/devel/appendixes/backup_barmanobjectstore" target="_blank" rel="noopener">object
  store appendix</a> until the templates move.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/n8n/templates/postgres-cnpg.yaml" target="_blank" rel="noopener"><code>platform/services/n8n/templates/postgres-cnpg.yaml</code></a>
  — a consumer `Cluster` + `ScheduledBackup` pair to copy settings from
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/n8n/values.yaml" target="_blank" rel="noopener"><code>platform/services/n8n/values.yaml</code></a>
  — `backup.destinationPath` and `backup.endpointURL` for one consumer
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/app-of-apps/values.yaml" target="_blank" rel="noopener"><code>platform/services/app-of-apps/values.yaml</code></a>
  — each `Application`'s sync policy
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/cloudnative-pg" target="_blank" rel="noopener"><code>platform/core/cloudnative-pg/</code></a>
  — the operator chart
