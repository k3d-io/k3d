#!/bin/bash

# Check if the user provided a release
if [ "$1" = "--release" ]; then
    shift
fi

# Assign the release variable
RELEASE=${1:-latest}

# Rest of the install script...
