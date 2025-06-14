#!/bin/sh
#

echo Linting Go Package...
cd subcalc-go && golangci-lint run && cd ..
echo Linting WASM binary for C@E...
cd subcalc-api && golangci-lint run
