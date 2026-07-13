#!/usr/bin/env bash
set -euo pipefail

version="${VERSION:-dev}"
root="$(cd "$(dirname "$0")/.." && pwd)"
dist="$root/dist"
stage="$dist/stage"

rm -rf "$dist"
mkdir -p "$stage"

targets=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
)

for target in "${targets[@]}"; do
  read -r goos goarch <<<"$target"
  name="ssh-mcp-${version}-${goos}-${goarch}"
  dir="$stage/$name"
  mkdir -p "$dir"
  binary="ssh-mcp"
  if [[ "$goos" == "windows" ]]; then
    binary="ssh-mcp.exe"
  fi
  (
    cd "$root"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
      -trimpath -ldflags="-s -w" -o "$dir/$binary" .
  )
  cp "$root/config.example.yaml" "$root/README.md" "$root/LICENSE" "$dir/"
  if [[ "$goos" == "windows" ]]; then
    (cd "$stage" && zip -qr "$dist/$name.zip" "$name")
  else
    (cd "$stage" && tar -czf "$dist/$name.tar.gz" "$name")
  fi
done

checksum() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$@"
  else
    shasum -a 256 "$@"
  fi
}

(
  cd "$dist"
  checksum ssh-mcp-*.tar.gz ssh-mcp-*.zip > checksums.txt
)

offline="$stage/ssh-mcp-${version}-offline"
mkdir -p "$offline"
cp -R "$stage"/ssh-mcp-${version}-darwin-* "$offline/"
cp -R "$stage"/ssh-mcp-${version}-linux-* "$offline/"
cp -R "$stage"/ssh-mcp-${version}-windows-* "$offline/"
cp "$root/docs/agent-install.md" "$offline/"
: > "$offline/checksums.txt"
for binary in "$offline"/ssh-mcp-${version}-*/ssh-mcp*; do
  relative="${binary#"$offline/"}"
  (cd "$offline" && checksum "$relative") >> "$offline/checksums.txt"
done
(cd "$stage" && zip -qr "$dist/ssh-mcp-${version}-offline.zip" "ssh-mcp-${version}-offline")
(
  cd "$dist"
  checksum "ssh-mcp-${version}-offline.zip" >> checksums.txt
)

rm -rf "$stage"
