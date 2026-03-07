#!/bin/bash

# only if redocly is not installed
if ! command -v redocly &> /dev/null
then
    echo "Redocly CLI not found, installing..."
    npm install -g @redocly/cli
    if [ $? -ne 0 ]; then
        echo "Failed to install Redocly CLI. Please install it manually and try again."
        exit 1
    fi
fi

# bundle the OpenAPI spec
redocly bundle