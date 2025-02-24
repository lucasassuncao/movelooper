# Use the Ubuntu base image
FROM ubuntu:latest

# Install basic tools
RUN apt-get update && \
    apt-get install -y \
    curl \
    wget \
    git \
    vim \
    nano \
    build-essential

SHELL ["/bin/bash", "-c"]

# Set working directory
WORKDIR /workspace