#!/usr/bin/env bash
set -euo pipefail

dockerfile="$1"
context="$2"
tag="$3"
shift 3

buildArgs=(--file "$dockerfile" --tag "$tag" "$@")
if [ -n "${MIRROR_REGISTRY:-}" ]; then
  buildArgs+=(--tag "${MIRROR_REGISTRY}/${tag}")
fi
if [ "${PUSH:-false}" = "true" ]; then
  buildArgs+=(--output type=registry,oci-mediatypes=true,compression=zstd,force-compression=true)
fi

docker buildx build "${buildArgs[@]}" "$context"
