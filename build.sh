#!/usr/bin/env bash
# Baut die Release-Archive nach dist/  –  ./build.sh 1.0.0
set -euo pipefail
cd "$(dirname "$0")"
v=${1:?Version angeben, z. B. 1.0.0}
rm -rf dist && mkdir -p dist
for t in "armv5 arm 5" "armv7 arm 7" "arm64 arm64 -" "amd64 amd64 -"; do
  set -- $t
  d=$(mktemp -d)
  (cd server && CGO_ENABLED=0 GOOS=linux GOARCH=$2 GOARM=${3#-} go build -trimpath -ldflags "-s -w -X main.version=$v" -o "$d/rackpanel" .)
  cp -r www "$d/www"
  tar -czf "dist/rackpanel-linux-$1.tar.gz" -C "$d" rackpanel www
  rm -rf "$d"
done
ls -la dist
