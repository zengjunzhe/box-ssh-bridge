#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
mkdir -p dist

for target in darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/amd64; do
  os=${target%/*}
  arch=${target#*/}
  suffix=""
  [ "$os" = windows ] && suffix=".exe"
  echo "building $os/$arch"
  GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION:-dev}" \
    -o "dist/box-ssh-bridge-${os}-${arch}${suffix}" \
    ./cmd/box-ssh-bridge
done
