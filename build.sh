#!/bin/bash

set -e

pushd ./client/
if [ ! -d "node_modules" ]; then
    echo "node_modules not found. Installing dependencies..."
    npm ci
fi

npm run build
popd

pushd rpicam
./build.sh
popd

go build
