#!/usr/bin/env bash
set -euo pipefail

profile="$1"

# renovate: datasource=docker depName=grafana/mcp-grafana
image="grafana/mcp-grafana:2.0.1"

root="$(git rev-parse --show-toplevel)"
file="$root/env.yaml"
if [ ! -f "$file" ]; then
  file="$root/env.example.yaml"
fi

if [ "$(yq ".grafana | has(\"$profile\")" "$file")" != "true" ]; then
  echo "No grafana.$profile profile in $file" >&2
  exit 1
fi

url="$(yq ".grafana.$profile.url" "$file")"
host="$(sed -E 's|^[a-z]+://([^/:]+).*|\1|' <<<"$url")"

# *.localhost resolves to the container itself, so point it at the VM, where Traefik is published.
hostArgs=()
if [[ "$host" == *.localhost ]]; then
  hostArgs=(--add-host "$host:host-gateway")
fi

exec docker run --rm -i "${hostArgs[@]}" \
  --env GRAFANA_URL="$url" \
  --env GRAFANA_USERNAME="$(yq ".grafana.$profile.username // \"\"" "$file")" \
  --env GRAFANA_PASSWORD="$(yq ".grafana.$profile.password // \"\"" "$file")" \
  --env GRAFANA_SERVICE_ACCOUNT_TOKEN="$(yq ".grafana.$profile.serviceAccountToken // \"\"" "$file")" \
  --volume /etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt:ro \
  "$image" --transport stdio --usage-stats=disabled
