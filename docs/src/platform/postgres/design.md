---
title: Postgres design
---

Every component that needs a relational database gets its own Postgres cluster from
<a href="https://cloudnative-pg.io/" target="_blank" rel="noopener">CloudNativePG</a> (CNPG), the
Kubernetes operator installed once for the whole platform.

## How it's set up

Each consumer's chart has a `postgres-cnpg.yaml` with a single-instance `Cluster` and a daily
`ScheduledBackup`. CNPG archives the WAL continuously and takes a gzip base backup every day to an
R2 bucket, under a path per consumer, kept for a few days (`retentionPolicy`). `backup.enabled` in
the values switches it per environment; it is off locally.

## Decisions and gotchas

- **Single instance, no standby**, a cost trade-off for a personal platform: backups are the
  recovery mechanism, not an extra on top of HA. See [Restore a backup](restore-a-backup.md).
- **One cluster per consumer**, so an app's database lives, is backed up and is restored with it.
- **Each consumer has its own bucket credentials**, from Vault through External Secrets, except
  Vault itself, which reads them from its bootstrap secret.
- **Data is not encrypted at rest**: Hetzner volumes have no encryption option.
- **Backups use CNPG's in-tree `barmanObjectStore`**, which upstream deprecated for the Barman Cloud
  plugin; the templates haven't moved yet.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/cloudnative-pg" target="_blank" rel="noopener"><code>platform/core/cloudnative-pg/</code></a>
  — the operator
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/n8n/templates/postgres-cnpg.yaml" target="_blank" rel="noopener"><code>n8n/templates/postgres-cnpg.yaml</code></a>
  — a consumer's `Cluster` and `ScheduledBackup`
