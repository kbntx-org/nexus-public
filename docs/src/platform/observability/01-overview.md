---
title: Overview
---

Observability in Nexus is one Helm chart,
<a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/monitoring" target="_blank" rel="noopener"><code>platform/core/monitoring/</code></a>,
built on the Grafana OSS stack rather than a vendor like Datadog. Two reasons: cost, and control
over visualization. Grafana mixes metrics, logs and arbitrary external sources (custom APIs
included) on one dashboard, instead of boxing panels into one vendor's query model.

Metrics and logs are stored in S3-compatible object storage (R2): it is less to operate than block
storage, and storage scales independently of the cluster. Locally, Loki falls back to the filesystem
and Mimir uses <a href="https://rustfs.com/" target="_blank" rel="noopener">RustFS</a>, an
S3-compatible store only Tilt deploys.

## Stack

| Signal     | Tool                                                                                          | Backing storage                                              |
| ---------- | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| Collection | <a href="https://grafana.com/docs/alloy/latest/" target="_blank" rel="noopener">Alloy</a>     | —                                                            |
| Metrics    | <a href="https://grafana.com/docs/mimir/latest/" target="_blank" rel="noopener">Mimir</a>     | S3-compatible (R2)                                           |
| Logs       | <a href="https://grafana.com/docs/loki/latest/" target="_blank" rel="noopener">Loki</a>       | S3-compatible (R2)                                           |
| Traces     | Not deployed yet                                                                              | —                                                            |
| Dashboards | <a href="https://grafana.com/docs/grafana/latest/" target="_blank" rel="noopener">Grafana</a> | CloudNativePG — see [Databases](../databases/01-overview.md) |

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/observability.svg"
</div>

## Collection

Alloy is the single agent for metrics and logs, deployed by Grafana's
<a href="https://github.com/grafana/k8s-monitoring-helm" target="_blank" rel="noopener"><code>k8s-monitoring</code></a>
chart rather than hand-written pipelines: the chart generates the Alloy configuration from feature
toggles and keeps only an allowlist of metrics per source, which keeps volume down. Stay on its
defaults and add a metric only when a dashboard needs it. Two exceptions exist for the dashboards:
the chart's broader node-exporter list (load averages, disk I/O), and the API server scrape, which
has no allowlist of its own and so keeps only the metrics its dashboard queries.

Two collectors run, because the work splits by scope: a DaemonSet on every node reads pod logs and
host metrics, and a single collector scrapes the cluster-level targets (kubelet, cAdvisor,
kube-state-metrics, API server). The node collector runs in its pod's network namespace, so its
network figures are the pod's, not the host's: that is why the node dashboard has no network panel.

## Metrics

Mimir runs from the
<a href="https://github.com/grafana/mimir/tree/main/operations/helm/charts/mimir-distributed" target="_blank" rel="noopener"><code>mimir-distributed</code></a>
chart with one replica per component, and keeps blocks in R2 for 15 days, like Loki. It's a PromQL
backend, so Grafana queries it as a Prometheus datasource. Its own Alertmanager and ruler are off:
alerting lives in Grafana (see [Alerts](#alerts)).

The chart defaults to Kafka-based ingest storage and zone-aware replication; both are switched off
in the values, since they buy availability this cluster doesn't need. The cost is single replicas:
for a minute or two after a store-gateway restart, queries reaching past the last few hours fail
until it is ready again.

## Logs

Loki runs in
<a href="https://grafana.com/docs/loki/latest/get-started/deployment-modes/#simple-scalable" target="_blank" rel="noopener">SimpleScalable</a>
mode with chunks and index in object storage. Alloy only ships log lines written after it starts
reading a file, so a restart never sends duplicates.

## Traces

Not wired up yet. The natural next piece is
<a href="https://grafana.com/oss/tempo/" target="_blank" rel="noopener">Tempo</a>, from the same
family and S3-backed like Loki. Traces matter here for things like CI/CD visibility (global data,
test failures).

## Dashboards and datasources

Both live in the chart, so Git is the source of truth. Dashboards are JSON files in
<a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/monitoring/dashboards" target="_blank" rel="noopener"><code>dashboards/</code></a>,
each loaded by Grafana's sidecar from a ConfigMap. There are no folders: a dashboard's title starts
with what it covers (`K8S - Nodes`, `Traefik - Ingress`), and the file is named after it. Each one
opens with a short text panel saying what it shows and how to read it. Datasources are declared in
the chart's `values.yaml`.

Provisioned dashboards can't be saved from the UI. To change one, edit it in Grafana locally, export
it as JSON (Share → Export, without "Export for sharing externally") and overwrite the file. Keep
its `uid`: links and bookmarks point to it.

## Alerts

Alerting goes through
<a href="https://grafana.com/docs/grafana/latest/alerting/" target="_blank" rel="noopener">Grafana's
own unified alerting</a> rather than a separate Alertmanager or `vmalert` — one less component to
run, and rules live next to the dashboards that inform them.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/monitoring" target="_blank" rel="noopener"><code>platform/core/monitoring/</code></a>
  — full monitoring Helm chart
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/rustfs" target="_blank" rel="noopener"><code>platform/core/rustfs/</code></a>
  — local-only S3-compatible object store
- [Databases](../databases/01-overview.md) — CNPG pattern backing Grafana's own state
- [Secrets](../secrets/01-overview.md) — how Loki's S3 credentials reach the cluster
