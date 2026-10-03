---
title: Overview
---

Observability in Nexus is one Helm chart,
<a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/monitoring" target="_blank" rel="noopener"><code>platform/core/monitoring/</code></a>,
built on the Grafana OSS stack rather than a vendor like Datadog. Two reasons: cost, and control
over visualization. Grafana mixes metrics, logs and arbitrary external sources (custom APIs
included) on one dashboard, instead of boxing panels into one vendor's query model.

Each signal is backed by S3-compatible object storage where possible: it is less to operate than
block storage, and storage scales independently of the cluster. Locally, Loki falls back to the
filesystem, and product services that need S3 get it from
<a href="https://rustfs.com/" target="_blank" rel="noopener">RustFS</a>, an S3-compatible store only
Tilt deploys.

## Stack

| Signal     | Tool                                                                                           | Backing storage                                              |
| ---------- | ---------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| Metrics    | <a href="https://docs.victoriametrics.com/" target="_blank" rel="noopener">VictoriaMetrics</a> | Local disk today — see [Metrics](#metrics)                   |
| Logs       | <a href="https://grafana.com/docs/loki/latest/" target="_blank" rel="noopener">Loki</a>        | S3-compatible (R2)                                           |
| Traces     | Not deployed yet                                                                               | —                                                            |
| Dashboards | <a href="https://grafana.com/docs/grafana/latest/" target="_blank" rel="noopener">Grafana</a>  | CloudNativePG — see [Databases](../databases/01-overview.md) |

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/observability.svg"
</div>

## Metrics

VictoriaMetrics (the
<a href="https://github.com/VictoriaMetrics/helm-charts/tree/master/charts/victoria-metrics-single" target="_blank" rel="noopener"><code>victoria-metrics-single</code></a>
chart) scrapes
<a href="https://github.com/prometheus/node_exporter" target="_blank" rel="noopener"><code>node-exporter</code></a>
(host signals) and
<a href="https://github.com/kubernetes/kube-state-metrics" target="_blank" rel="noopener"><code>kube-state-metrics</code></a>
(Kubernetes objects as metrics). It speaks Prometheus' scrape, remote-write and query APIs, so
PromQL, dashboards and exporters work unchanged. Only core metrics (CPU, I/O, memory, disk) are
covered so far, at node, pod and container level.

Metrics are the one signal still on a local volume. The plan is to move to
<a href="https://grafana.com/oss/mimir/" target="_blank" rel="noopener">Mimir</a> (Grafana OSS,
natively S3-backed, still PromQL) rather than stay single-node, and to replace the collection
DaemonSets with
<a href="https://grafana.com/docs/alloy/latest/" target="_blank" rel="noopener">Grafana Alloy</a>,
one agent for both metrics and logs.

## Logs

<a href="https://grafana.com/docs/loki/latest/clients/promtail/" target="_blank" rel="noopener">Promtail</a>
runs as a DaemonSet, tails container logs, attaches Kubernetes metadata and pushes to Loki, which
runs in
<a href="https://grafana.com/docs/loki/latest/get-started/deployment-modes/#simple-scalable" target="_blank" rel="noopener">SimpleScalable</a>
mode with chunks and index in object storage.

## Traces

Not wired up yet. The natural next piece is
<a href="https://grafana.com/oss/tempo/" target="_blank" rel="noopener">Tempo</a>, from the same
family and S3-backed like Loki. Traces matter here for things like CI/CD visibility (global data,
test failures).

## Adding a dashboard

- **Through the UI**, for iterating. The Grafana sidecar runs with `allowUiUpdates: true`, so
  dashboards saved interactively survive pod restarts.
- **Through the chart**, for anything that should be source of truth: the
  <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/monitoring/values.yaml" target="_blank" rel="noopener">monitoring
  values file</a> is reapplied on every ArgoCD sync. Promote UI-edited dashboards there eventually.

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
