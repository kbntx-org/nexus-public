---
title: Traefik
---

Traefik routes every request to your app's Service. Declare routes in the chart with an
`IngressRoute` and a cert-manager `Certificate`.

## Route traffic to an app

**Internal (WARP only), the default.** Copy
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/zot/templates/ingress.yaml" target="_blank" rel="noopener"><code>zot/templates/ingress.yaml</code></a>:

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: my-app-ingress
  labels:
    external-dns/enabled: 'true'
  annotations:
    external-dns.kubernetes.io/target: '{{ .Values.internalIngressTarget }}'
spec:
  routes:
    - match: Host(`{{ .Values.ingressHost }}`)
      kind: Rule
      services:
        - name: my-app
          port: http
  tls:
    secretName: my-app-tls
```

`internalIngressTarget` is Traefik's internal Service IP (`internalServiceClusterIP` in
`traefik/values.prod.yaml`): copy it into the app's values.

**Public.** Set `entryPoints: [websecure]` and drop the external-dns label and annotation, as the
docs site does in
<a href="https://github.com/kbntx-org/nexus/blob/main/docs/helm/templates/template.yaml" target="_blank" rel="noopener"><code>docs/helm/templates/template.yaml</code></a>.

Either way, add a `Certificate` from the `letsencrypt-dns01` `ClusterIssuer` for the host, behind
`certificate.enabled` (off locally, where the `*.localhost` certificate is the default), and let the
`traefik-ingress` namespace in on the container port in the app's network policy (see
`zot/templates/network-policy.yaml`).

## Debug a route

- **404 from Traefik:** the `match` doesn't fit the host, or the route sits on the wrong entry point
  (public routes need `websecure`).
- **Internal host doesn't resolve:** check the external-dns label and annotation, and that WARP is
  on.
- **Certificate not ready:** `kubectl describe certificate -n <namespace>`, then the challenge
  cert-manager created.

The Traefik dashboard and the **Traefik - Ingress** Grafana dashboard show routers and traffic.

How it works: [Traefik design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/traefik/values.prod.yaml" target="_blank" rel="noopener"><code>traefik/values.prod.yaml</code></a>
  — internal entry points

**External**:

- <a href="https://doc.traefik.io/traefik/routing/providers/kubernetes-crd/" target="_blank" rel="noopener">IngressRoute
  reference</a>
