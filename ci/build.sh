#!/bin/sh
#

env
echo Building C binary...
make
echo Building Go binary...
cd subcalc-go && make && cd ..
echo Building WASM binary for C@E...
cd subcalc-api && make
