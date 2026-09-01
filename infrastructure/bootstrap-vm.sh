#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${EUID}" -eq 0 ]]; then
  SUDO=""
  TARGET_USER="${SUDO_USER:-keelmesh}"
else
  SUDO="sudo"
  TARGET_USER="${USER}"
fi

export DEBIAN_FRONTEND=noninteractive

${SUDO} apt-get update
${SUDO} apt-get install -y \
  build-essential \
  ca-certificates \
  curl \
  gh \
  git \
  gnupg \
  jq \
  make \
  openssl \
  qemu-guest-agent \
  unzip \
  zip

${SUDO} install -m 0755 -d /etc/apt/keyrings
${SUDO} curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  -o /etc/apt/keyrings/docker.asc
${SUDO} chmod a+r /etc/apt/keyrings/docker.asc

source /etc/os-release
printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu %s stable\n' \
  "$(dpkg --print-architecture)" "${VERSION_CODENAME}" \
  | ${SUDO} tee /etc/apt/sources.list.d/docker.list >/dev/null

${SUDO} install -m 0755 -d /usr/share/keyrings
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg \
  | ${SUDO} tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null
printf 'deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main\n' \
  | ${SUDO} tee /etc/apt/sources.list.d/cloudflared.list >/dev/null

${SUDO} apt-get update
${SUDO} apt-get install -y \
  cloudflared \
  containerd.io \
  docker-buildx-plugin \
  docker-ce \
  docker-ce-cli \
  docker-compose-plugin

${SUDO} usermod -aG docker "${TARGET_USER}"
${SUDO} systemctl enable --now docker
${SUDO} systemctl enable --now qemu-guest-agent || true

${SUDO} install -d -m 0755 -o "${TARGET_USER}" -g "${TARGET_USER}" /srv/keelmesh
git config --global init.defaultBranch main

printf 'Docker: '
${SUDO} docker version --format '{{.Server.Version}}'
${SUDO} docker compose version
cloudflared --version
gh --version | head -n 1

printf '\nBootstrap complete. Sign out and back in before using Docker without sudo.\n'
