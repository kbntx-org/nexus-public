---
title: Traefik design
---

<a href="https://doc.traefik.io/traefik/" target="_blank" rel="noopener">Traefik</a> is the ingress
controller: it receives every request, public from the Cloudflare tunnel or private from WARP, and
routes it by host and path to an app's Service. Apps declare routes with Traefik's `IngressRoute`
CRD. <a href="https://cert-manager.io/" target="_blank" rel="noopener">cert-manager</a> issues their
certificates and
<a href="https://kubernetes-sigs.github.io/external-dns/" target="_blank" rel="noopener">external-dns</a>
writes their DNS records.

## How it's set up

Traefik has two kinds of entry point in production:

- **Public, `websecure`**: the tunnel forwards every platform hostname there. The tunnel's own
  wildcard DNS record covers public hosts, so they need no record of their own.
- **Internal, the default**: behind a second Service with a fixed cluster IP, reachable only over
  WARP. An internal route sets the `external-dns.kubernetes.io/target` annotation to that IP and the
  `external-dns/enabled=true` label, so its hostname resolves to it.

A route that sets no `entryPoints` is internal. Locally, routes use the default `web` and
`websecure` entry points, behind NodePorts kind publishes on 80 and 443.

## Decisions and gotchas

- **Private by default**, so forgetting `entryPoints` never exposes an app: the tunnel sends every
  hostname to `websecure`, which an internal route isn't attached to, so a public request to an
  internal host gets a 404.
- **Certificates come from Let's Encrypt through a DNS-01 challenge** on Cloudflare
  (`letsencrypt-dns01`), so no inbound HTTP endpoint is needed, and internal hosts get public
  certificates too.
- **external-dns only touches what it owns**: resources labeled `external-dns/enabled=true`, `A` and
  `CNAME` records, in the platform's zone.
- **Access logs keep only a small header allowlist.**
- **Forwarded headers are trusted from the pod network only**, where `cloudflared` runs.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/traefik" target="_blank" rel="noopener"><code>platform/core/traefik/</code></a>
  — entry points, internal Service, dashboard
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/cert-manager" target="_blank" rel="noopener"><code>platform/core/cert-manager/</code></a>
  — the `letsencrypt-dns01` issuer
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/external-dns/values.yaml" target="_blank" rel="noopener"><code>platform/core/external-dns/values.yaml</code></a>
  — sources, filters and record types
