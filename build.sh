#!/bin/bash

set -e

# Default values (false)
DEPS=false
UI=false
RUN=false

print_help() {
  echo "Usage: $0 [options]"
  echo
  echo "Options:"
  echo "  -deps, --dependencies       Install build dependencies"
  echo "  -ui, --front-end            Build UI front-end"
  echo "  -r, --run                   Run the program after build"
  echo "  -h, --help                  Show this help message"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    -deps|--dependencies)
      DEPS=true
      shift
      ;;
    -ui|--front-end)
      UI=true
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

if $DEPS; then
	sudo apt update
	sudo apt install -y \
		portaudio19-dev \
		libasound2-dev \
		libopus-dev \
		libopusfile-dev \
		rpicam-apps \
		libyuv-dev \
		cmake \
		libboost-program-options-dev \
		libdrm-dev \
		libexif-dev \
		meson \
		ninja-build

	echo "Installing golang..."
	wget https://go.dev/dl/go1.25.0.linux-arm64.tar.gz
	sudo rm -rf /usr/local/go
	sudo tar -C /usr/local -xzf go1.25.0.linux-arm64.tar.gz

	echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
	source ~/.bashrc

	go version
fi

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

go build

if $RUN; then
  ./baby-monitor
fi
