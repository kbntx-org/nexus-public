---
title: Vault design
---

<a href="https://developer.hashicorp.com/vault" target="_blank" rel="noopener">Vault</a> holds every
secret, and Git only declares which secret a workload needs. The
<a href="https://external-secrets.io/" target="_blank" rel="noopener">External Secrets Operator</a>
(ESO) copies it into a Kubernetes Secret at runtime, so a namespace can be rebuilt without losing a
credential.

## How it's set up

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/secrets.svg"
</div>

1. **Vault** runs in the cluster with a [CloudNativePG](../postgres/design.md) database as storage.
2. **Each app's chart** has a `ServiceAccount`, a `SecretStore` and an `ExternalSecret`.
3. **ESO** logs in to Vault's Kubernetes auth with that service account's token (audience `vault`);
   Vault checks it through the TokenReview API.
4. **The single `eso` role** reads only `platform/<namespace>` and `platform/<namespace>/*`, the
   caller's own namespace, so apps are isolated without a role each.
5. **ESO writes the Secret** and refreshes it on its interval; the pod reads it like any other.

## Decisions and gotchas

- **Postgres storage, not Raft**: the platform already backs up Postgres, so Vault needs no snapshot
  workflow of its own. Its schema is created once from `vault.sql`, only on a fresh cluster.
- **Manual unseal**: Vault starts sealed on every restart, keeping the unseal keys off the cluster.
  Until it is unsealed nothing new syncs, though existing Secrets stay.
- **`vault-secret` is created by hand** in the `vault` namespace (Postgres URL, superuser and backup
  keys), since Vault can't read its own storage credentials from itself.
- **ESO templates** reshape Vault values into whatever a consumer expects, for example a typed
  `basic-auth` Secret for CNPG (see `monitoring/templates/secrets.yaml`).
- **The Vault side is OpenTofu** (auth method, role, policy), an Atlantis stack in
  `external-secrets/provision/`.
- **Locally**, Vault runs in `-dev` mode in memory, seeded by `seed-script.yaml` into a `dev` mount
  with an `admin` role: no unseal, no setup.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/vault/server" target="_blank" rel="noopener"><code>platform/core/vault/server/</code></a>
  — Vault, its database and the local seed
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/external-secrets" target="_blank" rel="noopener"><code>platform/core/external-secrets/</code></a>
  — ESO chart, TokenReview RBAC and the `eso` role
