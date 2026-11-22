---
title: OpenTofu
---

Infrastructure outside the cluster is OpenTofu, planned and applied by Atlantis on pull requests.
Never run `tofu apply` from your machine unless Atlantis can't (see below).

## Change infrastructure

1. Open a PR and comment `atlantis plan` (autoplan is off): Atlantis plans the projects whose
   `when_modified` files changed and posts the plan. `atlantis plan -p <project>` plans one.
2. Comment `atlantis apply` once the plan looks right.
3. Merge, so `main` matches what is deployed.

## Add a stack

Add a project to `atlantis.yaml`, with the stack's files and the modules it calls in
`when_modified`. Put any new variable in Atlantis's Vault key as `TF_VAR_<variable>`. Pin
`required_version` to the `opentofu` version in `mise.toml`.

## Apply a plan that replaces a node (platform)

Atlantis runs on the cluster, so a `kubernetes` plan that replaces its node, or the control plane,
would kill the apply. Find its node with `kubectl get pod -n atlantis -o wide`. Then, from the PR
branch on your machine, with the state credentials and `TF_VAR_*` values from Atlantis's Vault key,
run `tofu apply`, comment `atlantis plan` to confirm there is no diff left, and merge.

How it works: [OpenTofu design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/blob/main/atlantis.yaml" target="_blank" rel="noopener"><code>atlantis.yaml</code></a>
  — projects
