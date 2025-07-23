#!/bin/sh

pushd ./client/
npm run build
popd

go run .
