---
title: Overview
---

Networking in Nexus is built on one rule: **the cluster has no open inbound ports.** The firewall
closes every inbound port and nodes reach the internet through a NAT gateway.
<a href="https://www.cloudflare.com/" target="_blank" rel="noopener">Cloudflare</a> sits in front, a
private <a href="https://www.hetzner.com/cloud" target="_blank" rel="noopener">Hetzner</a> network
sits behind, and traffic only gets in through outbound tunnels the cluster opens itself.

## Request path

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/traffic.svg"
</div>

1. **Cloudflare edge.** Cloudflare owns the DNS and proxies every request (WAF, DDoS protection,
   rate limiting), so the origin IP is never exposed.
2. **Tunnel.** In-cluster
   <a href="https://developers.cloudflare.com/cloudflare-one/connections/connect-networks" target="_blank" rel="noopener"><code>cloudflared</code></a>
   pods hold a persistent **outbound** connection to the edge: no inbound listener, no firewall
   hole, no public IP.
3. **Ingress.** The tunnel forwards the request over TLS to the ingress controller, which routes it
   in-cluster. Certificates come from
   <a href="https://cert-manager.io/" target="_blank" rel="noopener">cert-manager</a> and Let's
   Encrypt through a Cloudflare DNS-01 challenge, so no inbound HTTP endpoint is needed.

## Cloudflare-k8s-controller: the tunnel isn't hand-configured

Access rules used to live in Terraform, away from the workloads they protect. The repo's own
<a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/cloudflare-controller" target="_blank" rel="noopener"><code>cloudflare-controller</code></a>,
a small controller-runtime operator, moved them next to the workload: each chart declares its
Cloudflare objects as CRDs, and the controller reconciles them through the Cloudflare API.

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/cloudflare-controller.svg"
</div>

**Gating a hostname** is an `AccessApplication` plus the `AccessPolicy` resources it references, in
the consuming app's chart, not a Terraform or dashboard change. Policies must live in the
application's namespace, and the application stays not ready until each of them has been created on
Cloudflare.

## External-dns: DNS records aren't hand-managed either

<a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/external-dns" target="_blank" rel="noopener"><code>external-dns</code></a>
syncs Cloudflare DNS records from `Service`, `Ingress`, Traefik CRD and `DNSEndpoint` sources, so
records ship with the workloads they point to. It is the zone's only writer, which is why the
controller hands it a `DNSEndpoint` instead of calling the DNS API itself. It only acts on resources
labeled `external-dns/enabled=true`, and only manages `A`/`CNAME` records in the platform's own
zone.

## Private access via WARP

Operating the platform and reaching internal apps needs a way into the VPC without breaking the
no-inbound rule.
<a href="https://developers.cloudflare.com/cloudflare-one/connections/connect-devices/warp/" target="_blank" rel="noopener">Cloudflare
Zero Trust</a> with the WARP client uses the same outbound-only pattern as public traffic (the
private paths in the figure above). Two tunnels carry private traffic, each with its own scope:

- **The bastion's tunnel routes the entire VPC subnet**: node IPs and the bastion itself, mainly for
  SSH during non-Kubernetes operations. It is the one connector that can't live in the cluster,
  because the bastion is also the VPC's NAT gateway and must be reachable when nothing else is. Its
  <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/bastion" target="_blank" rel="noopener">Terraform</a>
  creates the tunnel and hands its token to cloud-init, which starts `cloudflared` as a container on
  first boot, so the token never leaves Terraform.

- **The cluster's `Tunnel` routes the pod and service CIDRs**, alongside public ingress. Private DNS
  records on Cloudflare can then point at private apps and databases, reachable directly from a
  WARP-connected device. The WARP default profile (Terraform, in
  <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/warp" target="_blank" rel="noopener"><code>platform/core/warp/</code></a>)
  keeps those ranges inside the tunnel through its split-tunnel excludes, and resolves
  cluster-internal domains against the cluster DNS through local-domain fallback.

**Gotcha: changing the bastion's config replaces the server.** Cloud-init only runs at creation. The
floating IP keeps the gateway address stable, but every private node loses egress until the new
server is up.

## SSH to a node

There is no "jump host with authorized keys". Every Hetzner server is a
<a href="https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/use-cases/ssh/ssh-infrastructure-access/" target="_blank" rel="noopener">Cloudflare
Zero Trust Infrastructure Access</a> target, gated by one policy: **identity** (an allow-listed
email) AND **device posture** (a "gateway" posture rule, i.e. the device is on WARP). Passing both
yields a short-lived SSH certificate signed by an account-wide CA that every node's `sshd` already
trusts.

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/ssh-access.svg"
</div>

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/cloudflare-controller" target="_blank" rel="noopener"><code>platform/core/cloudflare-controller/</code></a>
  — the `Tunnel`/`AccessApplication`/`AccessPolicy` operator
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/cloudflared/chart/templates/tunnel.yaml" target="_blank" rel="noopener"><code>platform/core/cloudflared/chart/templates/tunnel.yaml</code></a>
  — the cluster's own `Tunnel` resource (public ingress + private CIDR routes)
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/external-dns" target="_blank" rel="noopener"><code>platform/core/external-dns/</code></a>
  — Cloudflare DNS record automation
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/warp" target="_blank" rel="noopener"><code>platform/core/warp/</code></a>
  — WARP device profile, split-tunnel and local-domain-fallback config
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/bastion" target="_blank" rel="noopener"><code>platform/core/bastion/</code></a>
  — the bastion VM: NAT gateway, VPC-wide private-network tunnel, and the cloud-init that starts its
  tunnel
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/network" target="_blank" rel="noopener"><code>platform/core/network/</code></a>
  — the Hetzner VPC every private route ultimately targets
