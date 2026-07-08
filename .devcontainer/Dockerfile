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
    plocate && \
    rm -rf /var/lib/apt/lists/*
