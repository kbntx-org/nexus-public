---
title: Container registry
---

zot holds every image this repository builds and caches upstream images. Its UI and API are
reachable over WARP only, with credentials.

## Use an upstream image

Reference it as usual (`docker.io/...`, `ghcr.io/...`): the nodes pull it through the cache. To pull
a cached image yourself, prefix its path with `dockerhub/` or `ghcr/` on the registry host.

## Add a new built image

Name the image in the project's `build-ci` command, then add its repository to the first retention
policy in
<a href="https://github.com/kbntx-org/nexus/blob/main/platform/core/zot/values.yaml" target="_blank" rel="noopener"><code>values.yaml</code></a>,
or every tag it ever had stays in the bucket.

## Force a garbage collection (platform)

Restart the pod: the first sweep runs at start. Each sweep re-walks the whole bucket, so keep
`gcInterval` in hours; `gcDelay` must outlast the longest push, since layers upload before the
manifest.

How it works: [Container registry design](design.md).

## References

- <a href="https://github.com/kbntx-org/nexus/tree/main/platform/core/zot" target="_blank" rel="noopener"><code>platform/core/zot/</code></a>
  — chart, secrets and route
