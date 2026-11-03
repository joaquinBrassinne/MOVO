#!/usr/bin/env bash
# Copies the fonts and sprites of the Protomaps basemap into frontend/public/map, so the map
# does not load them from a third party (SDD 7, docs/work/fase-7-mapa.md).
#
#   scripts/map-assets.sh            # frontend/public/map (served by Next in development)
#   scripts/map-assets.sh <dir>      # any folder, e.g. one to upload to the bucket
#
# Source: github.com/protomaps/basemaps-assets, pinned to a commit so the result is reproducible.
# Fonts are Noto Sans (SIL OFL 1.1). The three stacks are the ones the basemap style uses.
set -euo pipefail

REPO=https://github.com/protomaps/basemaps-assets.git
COMMIT=028c18f713baecad011301ff7a69acc39bcc2ae7
SPRITE_FLAVOR=light

root=$(cd "$(dirname "$0")/.." && pwd)
dest=${1:-$root/frontend/public/map}

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

git -C "$work" init -q
git -C "$work" fetch -q --depth 1 "$REPO" "$COMMIT"
git -C "$work" checkout -q FETCH_HEAD

mkdir -p "$dest/fonts" "$dest/sprites/v4"
for stack in "Noto Sans Regular" "Noto Sans Medium" "Noto Sans Italic"; do
  rm -rf "${dest:?}/fonts/$stack"
  cp -r "$work/fonts/$stack" "$dest/fonts/$stack"
done
cp "$work"/sprites/v4/"$SPRITE_FLAVOR"*.json "$work"/sprites/v4/"$SPRITE_FLAVOR"*.png "$dest/sprites/v4/"
cp "$work/fonts/OFL.txt" "$dest/fonts/OFL.txt"

echo "map assets copied to $dest"
