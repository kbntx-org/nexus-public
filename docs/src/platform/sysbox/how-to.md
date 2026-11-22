---
title: Sysbox
---

Sysbox runs Docker inside a container without giving it control of the host.

## Run Docker in a pod

Set `runtimeClassName: sysbox-runc`, `hostUsers: false`, and a toleration and node selector for the
`workspace` pool; only its nodes run Sysbox. Copy
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/services/workspace/templates/stateful-set.yaml" target="_blank" rel="noopener"><code>workspace/templates/stateful-set.yaml</code></a>.
Put Docker's data folder on a volume, or it is deleted with the pod.

How it works: [Sysbox design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/sysbox" target="_blank" rel="noopener"><code>platform/core/sysbox/</code></a>
  — installer
