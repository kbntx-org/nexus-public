#!/usr/bin/env bash
set -euo pipefail

repository_url=https://github.com/kbntx-org/nexus.git

git config --global credential.helper \
  '!f() { echo "username=x-access-token"; echo "password=${GITHUB_TOKEN}"; }; f'

if [ ! -d "${CHECKOUT_PATH}/.git" ]; then
  git clone "${repository_url}" "${CHECKOUT_PATH}"
fi

cd "${CHECKOUT_PATH}"
mise trust
mise install
