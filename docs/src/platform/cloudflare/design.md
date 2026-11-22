---
title: Cloudflare design
---

<a href="https://www.cloudflare.com/" target="_blank" rel="noopener">Cloudflare</a> is how anything
reaches the cluster, which has **no open inbound ports**: the firewall closes them all, and nodes
reach the internet through the bastion's NAT. Public requests and private access both ride outbound
tunnels the platform opens itself.

## How it's set up

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/traffic.svg"
</div>

1. **The edge** owns DNS and proxies public requests (WAF, DDoS protection, rate limiting), so the
   origin IP never shows.
2. **In-cluster `cloudflared` pods** hold an outbound tunnel to the edge and forward requests over
   TLS to [Traefik](../traefik/design.md).
3. **Private access** goes through
   <a href="https://developers.cloudflare.com/cloudflare-one/connections/connect-devices/warp/" target="_blank" rel="noopener">WARP</a>
   and two tunnels: the cluster's tunnel routes the pod and service ranges, so a WARP device reaches
   internal apps and databases directly; the bastion's tunnel routes the whole VPC, for SSH to the
   nodes.

The repository's own `cloudflare-controller` reconciles three CRDs through the Cloudflare API, so
each chart declares its Cloudflare objects next to its workload instead of in OpenTofu:

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/cloudflare-controller.svg"
</div>

- **`Tunnel`** creates the tunnel, runs its `cloudflared` Deployment, routes its private ranges and
  publishes its hostnames as a `DNSEndpoint` for external-dns.
- **`AccessApplication`** and **`AccessPolicy`** put Cloudflare Access login in front of a hostname.

## Decisions and gotchas

- **external-dns is the zone's only writer**, which is why the controller hands it a `DNSEndpoint`
  instead of calling the DNS API itself.
- **The bastion's `cloudflared` can't live in the cluster**: the bastion is the VPC's NAT gateway
  and must be reachable when nothing else is. OpenTofu creates its tunnel and passes the token to
  cloud-init, so the token never leaves OpenTofu.
- **Changing the bastion's config replaces the server**, since cloud-init only runs at creation. The
  floating IP keeps the gateway address, but every node loses egress until the new server is up.
- **WARP reaches Services directly, past Traefik.** A login on an `IngressRoute` doesn't protect the
  `Service` behind it: give such a UI a network policy that only lets its login proxy in.
- **SSH has no jump host or authorized keys.** Each server is an Access for Infrastructure target:
  an allow-listed email on a WARP device gets a short-lived certificate from an account-wide CA that
  every node's `sshd` trusts.

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/ssh-access.svg"
</div>

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/cloudflare-controller" target="_blank" rel="noopener"><code>platform/core/cloudflare-controller/</code></a>
  — the `Tunnel`, `AccessApplication` and `AccessPolicy` operator
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/cloudflared/chart/templates/tunnel.yaml" target="_blank" rel="noopener"><code>cloudflared/chart/templates/tunnel.yaml</code></a>
  — the cluster's tunnel
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/warp" target="_blank" rel="noopener"><code>platform/core/warp/</code></a>
  — WARP profile, split tunnel and local domain fallback
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/bastion" target="_blank" rel="noopener"><code>platform/core/bastion/</code></a>
  — NAT gateway, VPC tunnel and SSH access targets
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/network" target="_blank" rel="noopener"><code>platform/core/network/</code></a>
  — the private network
