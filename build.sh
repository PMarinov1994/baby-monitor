#!/bin/bash

set -e

# Default values (false)
UI=false
NCAM=false
RUN=false

print_help() {
  echo "Usage: $0 [options]"
  echo
  echo "Options:"
  echo "  -ui, --front-end            Enable front-end placeholder"
  echo "  -ncam, --native-camera-lib  Enable native camera lib placeholder"
  echo "  -r, --run                   Run the if/else block"
  echo "  -h, --help                  Show this help message"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    -ui|--front-end)
      UI=true
      shift
      ;;
    -ncam|--native-camera-lib)
      NCAM=true
      shift
      ;;
    -r|--run)
      RUN=true
      shift
      ;;
    -h|--help)
      print_help
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use --help for usage information."
      exit 1
      ;;
  esac
done

# Respond to flags
if $UI; then
	pushd ./client/
	if [ ! -d "node_modules" ]; then
	    echo "node_modules not found. Installing dependencies..."
	    npm ci
	fi

	npm run build
	popd
fi

if $NCAM; then
	pushd rpicam/native

	mkdir -p _build
	pushd _build

	cmake -G Ninja -DCMAKE_BUILD_TYPE=Debug -DCMAKE_EXPORT_COMPILE_COMMANDS=ON ..
	cmake --build .

	popd
	popd
fi

if $RUN; then
  go run .
else
	go build
fi
