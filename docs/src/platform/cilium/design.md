---
title: Cilium design
---

<a href="https://cilium.io/" target="_blank" rel="noopener">Cilium</a> is the cluster's network
plugin. Its eBPF dataplane also replaces kube-proxy and enforces network policies, instead of a CNI
plus a separate proxy layer and policy controller.

## How it's set up

Cilium is installed first on a fresh cluster, since nothing schedules without a CNI, and then
managed by ArgoCD. k3s runs with `disable-kube-proxy` and `disable-network-policy`.
<a href="https://docs.cilium.io/en/stable/observability/hubble/" target="_blank" rel="noopener">Hubble</a>
shows the traffic and what policies drop.

Every chart with a workload outside `kube-system` ships `templates/network-policy.yaml`, switched by
`networkPolicy.enabled`, with both `Ingress` and `Egress` in `policyTypes`, so a pod accepts and
sends only what its policy allows:

- **Ingress** from `traefik-ingress` on the container port for routed apps, from `cloudflared` for
  what WARP devices reach directly (databases, SSH), and from the same namespace for multi-pod apps.
- **Egress** to `kube-dns`, then only the peers and internet ports the workload needs. "The
  internet" is every address minus the private and link-local ranges, so no pod reaches the VPC, the
  bastion or the Hetzner metadata endpoint through it.

## Decisions and gotchas

- **Standard `NetworkPolicy`, not `CiliumNetworkPolicy`**, so the same policies hold in kind, where
  Cilium can't run (it needs BPF privileges a Sysbox container lacks).
- **Host-network peers are matched by port alone.** The API server, the kubelet or a node pulling
  through Traefik have node IPs, which an `ipBlock` matches only on Cilium with
  `policyCIDRMatchMode: nodes`. A rule with `ports` and no peer works on every CNI.
- **There is no deny rule.** A second policy on the same pod widens what it reaches: keep one policy
  per pod.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/cilium" target="_blank" rel="noopener"><code>platform/core/cilium/</code></a>
  — Cilium and Hubble
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/zot/templates/network-policy.yaml" target="_blank" rel="noopener"><code>zot/templates/network-policy.yaml</code></a>
  — an example policy
