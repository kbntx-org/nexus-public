---
title: Sysbox design
---

<a href="https://github.com/nestybox/sysbox" target="_blank" rel="noopener">Sysbox</a> is a
container runtime that lets a container run its own Docker daemon, even a kind cluster, inside a
user namespace. The alternatives, `privileged: true` or a mounted Docker socket, hand the container
root on the host.

## How it's set up

- **In the cluster**, ArgoCD deploys an installer DaemonSet on nodes labeled `sysbox-install=yes`.
  It registers a `sysbox-runc` `RuntimeClass` that pods opt into with `runtimeClassName`. Only the
  tainted `workspace` pool carries the label, so no production node changes runtime.
- **In the local VM**, Colima's boot scripts install it, so an autonomous task gets a whole
  environment of its own (see
  [Run an isolated environment](../local-environment/how-to.md#run-an-isolated-environment)).

## Decisions and gotchas

- **We build the installer image** because upstream publishes it for amd64 only and its script
  needed patching for k3s.
- **It accepts only listed Kubernetes minors** (`is_supported_k8s_version()`), so a k3s minor
  upgrade waits for it (see [Upgrade k3s](../kubernetes/how-to.md#upgrade-k3s)).
- **Its containerd drop-in turns off unprivileged ports and ICMP** on those nodes: the sysctls
  containerd sets in every pod sandbox can't be written under Sysbox, and `sysbox-runc` pods would
  fail to start.

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/sysbox" target="_blank" rel="noopener"><code>platform/core/sysbox/</code></a>
  — installer image, patched script, containerd drop-in and DaemonSet chart
- <a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/local/colima.yaml" target="_blank" rel="noopener"><code>platform/core/local/colima.yaml</code></a>
  — the VM's Sysbox install
