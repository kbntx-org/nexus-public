---
title: Infrastructure changes
---

<a href="https://www.runatlantis.io/" target="_blank" rel="noopener">Atlantis</a> runs OpenTofu from
pull request comments. We chose it over GitHub Actions jobs because it locks a stack per PR and
applies exactly the plan that was reviewed.

<div class="nexus-diagram">
--8<-- "src/assets/diagrams/atlantis.svg"
</div>

## How it works

1. Comment `atlantis plan` (or `atlantis plan -p <project>`) on a PR; Atlantis posts the plan.
2. Comment `atlantis apply` once the plan looks right. There are no apply requirements: as a solo
   maintainer, no review could ever satisfy them.
3. Merge, so `main` matches what is deployed.

Every stack goes through Atlantis except the cluster nodes (`platform/core/kubernetes/provision`),
applied from a laptop: Atlantis runs on them, and replacing its node would kill the apply.

## Credentials

Atlantis reads everything from the Vault secret `platform/atlantis`: the GitHub App's `privateKey`
and `webhookSecret`, and every other property loaded as an environment variable (R2 state
credentials, and stack variables as `TF_VAR_<variable>`).

## Adding a stack

Add a project to `atlantis.yaml`, with the stack's files and the modules it calls in
`when_modified`, and any new variable to `platform/atlantis`. Pin the stack's `required_version` to
the exact `opentofu` version in `mise.toml`: Atlantis runs that version, and Renovate bumps both in
one PR.

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/atlantis.yaml" target="_blank" rel="noopener"><code>atlantis.yaml</code></a>
  — projects Atlantis plans
- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/services/atlantis" target="_blank" rel="noopener"><code>platform/services/atlantis/</code></a>
  — Atlantis chart, server-side repo config, routes and secrets
