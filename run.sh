#!/bin/bash

pushd ./client/
npm run build
popd

go run .
