#!/bin/sh
# Serve this directory over HTTP. WebAssembly.instantiateStreaming needs the
# correct MIME type, so opening index.html from the filesystem will not work.
set -e
cd "$(dirname "$0")"
GOOS=js GOARCH=wasm go build -o main.wasm .
exec python3 -m http.server "${1:-8000}"
