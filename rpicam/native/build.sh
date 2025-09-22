#!/bin/bash

set -e

mkdir -p _build
cd _build

cmake -G Ninja -DCMAKE_BUILD_TYPE=Debug -DCMAKE_EXPORT_COMPILE_COMMANDS=ON ..
cmake --build .
