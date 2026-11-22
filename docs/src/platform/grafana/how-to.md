---
title: Grafana
---

Grafana shows every metric and log of the platform. In production it is reachable over WARP;
locally, start `monitoring` in Tilt (`tilt up -- monitoring`) and log in with the seeded `admin` /
`admin`.

## Add a dashboard

1. Build it in the local Grafana, then export it as JSON (**Share → Export**, without "Export for
   sharing externally").
2. Save it in the `dashboards/` folder of the chart that owns it, named after its title. The title
   starts with what it covers (`Traefik - Ingress`); there are no folders.
3. Render the folder into a ConfigMap labelled `grafana-dashboard: "1"`, as
   <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/traefik/templates/config-map.yaml" target="_blank" rel="noopener"><code>traefik/templates/config-map.yaml</code></a>
   does.
4. Start it with a short text panel saying what it shows and how to read it.

To change one, edit it locally, export it and overwrite the file. Keep its `uid`: links point to it.

## Scrape a new component

Add a `ServiceMonitor` or `PodMonitor` to its chart, and allow Alloy in through its network policy.
Keep only the metrics your dashboards query (`metricRelabelings`), as the ArgoCD values do for its
components.

## Add a datasource (platform)

Without a secret, add it to the monitoring chart's `values.yaml`. With one, add an `ExternalSecret`
to the owning chart whose template renders the datasource YAML into a Secret labelled
`grafana-datasource: "1"`, so the credential never touches Git.

## Find logs

In **Explore**, pick Loki and filter by `namespace` and `pod`. Only lines written after Alloy
started reading a file are shipped.

How it works: [Grafana design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/monitoring/values.yaml" target="_blank" rel="noopener"><code>monitoring/values.yaml</code></a>
  — collection, retention, datasources
