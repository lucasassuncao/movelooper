FROM ubuntu:26.04

# Install basic tools and apply security updates
RUN apt-get update && \
    apt-get -y upgrade && \
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
