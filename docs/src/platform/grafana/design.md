---
title: Grafana design
---

Observability is the Grafana OSS stack in one chart:
<a href="https://grafana.com/docs/alloy/latest/" target="_blank" rel="noopener">Alloy</a> collects,
<a href="https://grafana.com/docs/mimir/latest/" target="_blank" rel="noopener">Mimir</a> stores
metrics, <a href="https://grafana.com/docs/loki/latest/" target="_blank" rel="noopener">Loki</a>
stores logs and
<a href="https://grafana.com/docs/grafana/latest/" target="_blank" rel="noopener">Grafana</a> shows
them. We chose it over a vendor such as Datadog for cost, and because Grafana mixes metrics, logs
and any other source on one dashboard. There are no traces yet.

## How it's set up

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/observability.svg"
</div>

1. **Alloy**, from Grafana's `k8s-monitoring` chart, runs as a DaemonSet for pod logs and host
   metrics, plus one collector for cluster-level targets (kubelet, cAdvisor, kube-state-metrics, API
   server) and every `ServiceMonitor` and `PodMonitor`.
2. **Mimir** and **Loki** keep their data in R2 for 15 days. Locally Loki uses the filesystem and
   Mimir a local S3 store, RustFS.
3. **Grafana**, with its own CNPG database, loads every dashboard and datasource it finds in any
   namespace by label, so each chart ships its own next to its workload.

## Decisions and gotchas

- **Object storage, not volumes**: less to operate, and it grows apart from the cluster.
- **Only allowlisted metrics are kept**, from the chart's defaults, to keep volume down. Add a
  metric only when a dashboard needs it.
- **One replica per component, no Kafka ingest or zone replication**: availability this cluster
  doesn't need. After a store-gateway restart, queries older than a few hours fail for a minute or
  two.
- **Mimir's Alertmanager and ruler are off**: alerts belong in Grafana's own alerting. No alert
  rules exist yet.
- **The node collector sees its pod's network**, not the host's, so the node dashboard has no
  network panel.
- **Provisioned dashboards can't be saved from the UI**: Git is the source.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/monitoring" target="_blank" rel="noopener"><code>platform/core/monitoring/</code></a>
  — the chart, cluster-wide dashboards and datasources
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/rustfs" target="_blank" rel="noopener"><code>platform/core/rustfs/</code></a>
  — local-only S3 store
