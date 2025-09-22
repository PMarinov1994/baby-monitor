#!/bin/bash

set -e

mkdir -p _build
cd _build

cmake -G Ninja -DCMAKE_EXPORT_COMPILE_COMMANDS=ON ..
ninja
