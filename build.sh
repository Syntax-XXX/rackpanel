#!/usr/bin/env bash
# Builds the release archives into dist/  –  ./build.sh 1.2.0 [rackbox-version]
#   rackpanel-linux-<arch>.tar.gz  (rackpanel + www/)
#   rackbox-linux-<arch>.tar.gz    (rackbox + addon/ for RackPanel)
set -euo pipefail
cd "$(dirname "$0")"
v=${1:?version, e.g. 1.2.0}; rbv=${2:-0.1.0}
rm -rf dist && mkdir -p dist
for t in "armv5 arm 5" "armv7 arm 7" "arm64 arm64 -" "amd64 amd64 -"; do
  set -- $t
  d=$(mktemp -d)
  (cd server && CGO_ENABLED=0 GOOS=linux GOARCH=$2 GOARM=${3#-} go build -trimpath -ldflags "-s -w -X main.version=$v" -o "$d/rackpanel" .)
  cp -r www "$d/www"
  tar -czf "dist/rackpanel-linux-$1.tar.gz" -C "$d" rackpanel www
  rm -rf "$d" && d=$(mktemp -d)
  (cd rackbox && CGO_ENABLED=0 GOOS=linux GOARCH=$2 GOARM=${3#-} go build -trimpath -ldflags "-s -w -X main.version=$rbv" -o "$d/rackbox" .)
  cp -r rackbox/addon "$d/addon"
  tar -czf "dist/rackbox-linux-$1.tar.gz" -C "$d" rackbox addon
  rm -rf "$d"
done
ls -la dist
