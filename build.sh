#!/bin/bash

set -e

# Default values (false)
DEPS=false
UI=false
NCAM=false
RUN=false

print_help() {
  echo "Usage: $0 [options]"
  echo
  echo "Options:"
  echo "  -deps, --dependencies       Enable front-end placeholder"
  echo "  -ui, --front-end            Enable front-end placeholder"
  echo "  -ncam, --native-camera-lib  Enable native camera lib placeholder"
  echo "  -r, --run                   Run the if/else block"
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

if $DEPS; then
	sudo apt update
	sudo apt install -y \
		portaudio19-dev \
		libasound2-dev \
		libopus-dev \
		libopusfile-dev \
		rpicam-apps \
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

if $NCAM; then
	pushd rpicam/native

	mkdir -p _build
	pushd _build

	cmake -G Ninja -DCMAKE_INSTALL_PREFIX=/usr -DCMAKE_BUILD_TYPE=Release -DCMAKE_EXPORT_COMPILE_COMMANDS=ON ..
	cmake --build .
	sudo cmake --install .

	popd
	popd
fi

if $RUN; then
  go run .
else
	go build
fi
