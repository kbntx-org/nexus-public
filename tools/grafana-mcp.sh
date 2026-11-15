#!/usr/bin/env bash
set -euo pipefail

profile="$1"
port="$2"

# renovate: datasource=docker depName=grafana/mcp-grafana
image="grafana/mcp-grafana:2.0.1"
containerPort="8000"

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

# Exported and passed to docker by name only, so the credentials never appear on a command line.
export GRAFANA_URL="$url"
GRAFANA_USERNAME="$(yq ".grafana.$profile.username // \"\"" "$file")"
GRAFANA_PASSWORD="$(yq ".grafana.$profile.password // \"\"" "$file")"
GRAFANA_SERVICE_ACCOUNT_TOKEN="$(yq ".grafana.$profile.serviceAccountToken // \"\"" "$file")"
export GRAFANA_USERNAME GRAFANA_PASSWORD GRAFANA_SERVICE_ACCOUNT_TOKEN

exec docker run --rm "${hostArgs[@]}" \
  --publish "127.0.0.1:$port:$containerPort" \
  --env GRAFANA_URL \
  --env GRAFANA_USERNAME \
  --env GRAFANA_PASSWORD \
  --env GRAFANA_SERVICE_ACCOUNT_TOKEN \
  --volume /etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt:ro \
  "$image" --transport streamable-http --address "0.0.0.0:$containerPort" --allowed-hosts "localhost:$port" \
  --usage-stats=disabled
