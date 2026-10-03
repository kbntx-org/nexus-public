---
title: Overview
---

Every component that needs a relational database gets its own Postgres cluster from
<a href="https://cloudnative-pg.io/" target="_blank" rel="noopener">CloudNativePG</a> (CNPG), the
Kubernetes-native Postgres operator.

## The shared shape

Each consumer's chart declares the same pair in a `postgres-cnpg.yaml` template: a single-instance
`postgresql.cnpg.io/v1` `Cluster` and a daily `ScheduledBackup` targeting it. Mostly the database
name, credentials secret and storage size differ, so copy the closest existing template (Vault's
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/vault/server/templates/postgres-cnpg.yaml" target="_blank" rel="noopener"><code>postgres-cnpg.yaml</code></a>,
for example) rather than writing one from scratch.

**Every cluster is single-instance.** There is no standby replica to fail over to: a deliberate cost
trade-off for a personal platform. Backups are therefore the actual recovery mechanism, not an extra
on top of HA.

## Backups

Backups go through CNPG's
<a href="https://cloudnative-pg.io/documentation/current/backup_recovery/" target="_blank" rel="noopener">barman
integration</a>: continuous WAL archiving plus the `ScheduledBackup`'s daily base backup,
gzip-compressed and pushed to an R2 bucket under a per-consumer path.

- **Retention** is a few days, set by the `Cluster`'s `backup.retentionPolicy`.
- **Toggle:** `backup.enabled` in the chart values, per environment; off in local development.
- **Bucket credentials** come from each consumer's own secret, never a shared one: Vault-backed
  through ESO, except for Vault itself, which reads them from its
  [bootstrap secret](../secrets/01-overview.md#vault-in-the-cluster).

To get data back, follow [Restore a CNPG backup](../../runbooks/01-restore-a-cnpg-backup.md).

## Encryption at rest

**Postgres data is not encrypted at rest.** Hetzner block storage has no native encryption option,
so the PVC behind each `Cluster` is plain. Kubernetes `Secret` objects are encrypted (see
[Secrets](../secrets/01-overview.md#encryption-at-rest)); don't assume a Postgres volume is equally
protected.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/cloudnative-pg" target="_blank" rel="noopener"><code>platform/core/cloudnative-pg/</code></a>
  — the operator chart, installed once, used by every consumer
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/vault/server/templates/postgres-cnpg.yaml" target="_blank" rel="noopener"><code>platform/core/vault/server/templates/postgres-cnpg.yaml</code></a>
  — an example consumer `Cluster` + `ScheduledBackup` pair
- [Secrets](../secrets/01-overview.md) — where each consumer's backup/DB credentials actually live
