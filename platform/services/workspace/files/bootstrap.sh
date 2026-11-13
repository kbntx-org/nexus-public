#!/usr/bin/env bash
set -euo pipefail

repository_url=https://github.com/kbntx-org/nexus.git
port=8080

ssh_directory="${HOME}/.ssh"
ssh_host_key="${ssh_directory}/ssh_host_ed25519_key"

mkdir -p "${ssh_directory}"
chmod 700 "${ssh_directory}"
printf '%s\n' "${SSH_AUTHORIZED_KEYS:?}" >"${ssh_directory}/authorized_keys"
printf '%s\n' "${SSH_HOST_KEY:?}" >"${ssh_host_key}"
chmod 600 "${ssh_directory}/authorized_keys" "${ssh_host_key}"
sudo mkdir -p /run/sshd
sudo /usr/sbin/sshd

git config --global credential.helper \
  '!f() { echo "username=x-access-token"; echo "password=${GITHUB_TOKEN}"; }; f'

if [ ! -d "${CHECKOUT_PATH}/.git" ]; then
  git clone "${repository_url}" "${CHECKOUT_PATH}"
fi

cd "${CHECKOUT_PATH}"
mise trust
mise install

mkdir -p "${HOME}/.docker/cli-plugins"
ln -sf "$(mise which docker-compose)" "${HOME}/.docker/cli-plugins/docker-compose"
ln -sf "$(mise which docker-buildx)" "${HOME}/.docker/cli-plugins/docker-buildx"

exec code-server --bind-addr "0.0.0.0:${port}" --auth password "${CHECKOUT_PATH}"
