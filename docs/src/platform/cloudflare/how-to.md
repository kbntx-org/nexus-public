---
title: Cloudflare
---

Cloudflare carries every public request and every private connection to the cluster. To route a
hostname to an app, see [Traefik](../traefik/how-to.md).

## Connect to private apps

Install the WARP client and enroll it in the Zero Trust organization with an allow-listed email.
Internal hostnames, cluster DNS names and pod and service IPs then resolve and connect directly.

## Put a login in front of a hostname

Add an `AccessApplication` for the hostname and the `AccessPolicy` resources it references to the
app's chart, all in the app's namespace. The application stays not ready until each policy exists on
Cloudflare (`kubectl get accessapplication -n <namespace>`).

## SSH to a node

With WARP on, `ssh engineer@<node-private-ip>`. Cloudflare checks your identity and WARP posture and
issues a short-lived certificate; there are no keys to manage.

## Change the bastion (platform)

Its OpenTofu change replaces the server, so every node loses egress for a few minutes: plan it at a
quiet time (see [OpenTofu](../opentofu/how-to.md)).

How it works: [Cloudflare design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/cloudflare-controller/api/v1alpha1" target="_blank" rel="noopener"><code>cloudflare-controller/api/v1alpha1/</code></a>
  — CRD fields
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/bastion/ssh-access.tf" target="_blank" rel="noopener"><code>bastion/ssh-access.tf</code></a>
  — SSH targets and policy
