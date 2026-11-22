---
title: Cilium
---

Cilium is the network plugin and enforces every chart's network policy.

## Write a network policy

Copy the closest existing `templates/network-policy.yaml` (a routed app, a database, a multi-pod
app) and keep one policy per pod. Wrap it in `{{- if .Values.networkPolicy.enabled }}`, set
`networkPolicy.enabled: true` in `values.yaml` and `false` in `values.local.yaml`. Allow DNS to
`kube-dns`, then only what the workload talks to.

## Find what a policy drops

In production, over WARP:

```sh
hubble observe --verdict DROPPED --namespace <namespace>
```

or the `hubble_drop_total` and `hubble_policy_verdicts_total` metrics in Grafana. A drop to a node
IP (API server, kubelet) needs a port-only rule, see
[Cilium design](design.md#decisions-and-gotchas).

How it works: [Cilium design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/cilium" target="_blank" rel="noopener"><code>platform/core/cilium/</code></a>
  — Cilium values and Hubble route
