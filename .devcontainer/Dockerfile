FROM ubuntu:24.04

# Install basic tools and apply security updates
RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y --no-install-recommends \
    curl \
    wget \
    vim \
    nano \
    less \
    build-essential \
    ca-certificates \
    strace \
    lsof \
    iproute2 \
    dnsutils \
    htop \
    ripgrep \
    plocate \
    gnupg && \
    rm -rf /var/lib/apt/lists/*

# Install VHS (https://github.com/charmbracelet/vhs) and its ttyd/ffmpeg dependencies
# via Charm's apt repository
RUN mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://repo.charm.sh/apt/gpg.key | gpg --dearmor -o /etc/apt/keyrings/charm.gpg && \
    echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" > /etc/apt/sources.list.d/charm.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends vhs ttyd ffmpeg && \
    rm -rf /var/lib/apt/lists/*

# VHS drives a headless Chromium (downloaded on first run by go-rod) to render
# the recording; install its runtime shared libraries so that browser can start
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    libnss3 \
    libnspr4 \
    libatk1.0-0t64 \
    libatk-bridge2.0-0t64 \
    libcups2t64 \
    libdrm2 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libpango-1.0-0 \
    libcairo2 \
    libasound2t64 \
    libatspi2.0-0t64 \
    libxshmfence1 \
    libglib2.0-0t64 && \
    rm -rf /var/lib/apt/lists/*
