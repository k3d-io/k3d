#!/bin/bash

# Check if version is provided
if [ -n "$1" ]; then
  VERSION=$1
else
  # Get the latest version
  VERSION=$(curl -s https://api.github.com/repos/k3d-io/k3d/releases/latest | grep -oP '(?<=tag_name": ")[^"]*')
fi

# Download and install the specified version
curl -s -o /tmp/k3d https://github.com/k3d-io/k3d/releases/download/$VERSION/k3d-$VERSION-linux-amd64
chmod +x /tmp/k3d
sudo mv /tmp/k3d /usr/local/bin/k3d