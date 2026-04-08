FROM ubuntu:24.04

# Install basic tools
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    curl \
    wget \
    vim \
    nano \
    build-essential \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /workspace