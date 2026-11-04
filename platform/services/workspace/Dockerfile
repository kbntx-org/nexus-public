# syntax=docker/dockerfile:1.27

FROM debian:trixie-slim

ARG DOCKER_VERSION
ARG MISE_VERSION
# renovate: datasource=github-releases depName=coder/code-server
ARG CODE_SERVER_VERSION=4.144.0

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl git iptables sudo tmux xz-utils \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL "https://download.docker.com/linux/static/stable/$(uname -m)/docker-${DOCKER_VERSION}.tgz" \
    | tar -xz --strip-components=1 -C /usr/local/bin

RUN curl -fsSL https://code-server.dev/install.sh | sh -s -- --version "${CODE_SERVER_VERSION}"

RUN useradd --create-home --uid 1000 --shell /bin/bash workspace \
    && echo "workspace ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/workspace \
    && chmod 0440 /etc/sudoers.d/workspace

USER workspace
WORKDIR /home/workspace

RUN curl -fsSL https://mise.run | MISE_VERSION="v${MISE_VERSION}" sh

RUN echo 'eval "$(mise activate bash)"' >> /home/workspace/.bashrc

ENV PATH="/home/workspace/.local/bin:/home/workspace/.local/share/mise/shims:${PATH}"
