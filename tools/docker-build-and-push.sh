#!/usr/bin/env bash
set -euo pipefail

dockerfile="$1"
context="$2"
shift 2

buildArgs=(--file "$dockerfile" "$@")
if [ "${PUSH:-false}" = "true" ]; then
  buildArgs+=(--output type=registry,oci-mediatypes=true,compression=zstd,force-compression=true)
fi

docker buildx build "${buildArgs[@]}" "$context"
