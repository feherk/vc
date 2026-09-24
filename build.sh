#!/usr/bin/env bash
# vc build: minden platform a dist/ mappába, a verzió a main.go-ból.
#
#   ./build.sh            mind a 3 release-asset (darwin arm64, linux amd64, windows amd64)
#   ./build.sh --local    csak linux-amd64 (gyors fejlesztői kör)
#   ./build.sh --install  build + a helyi vc cseréje (~/.local/bin/vc, előtte vc.bak mentés)
#   ./build.sh --rollback a vc.bak visszaállítása
#   ./build.sh --no-notarize aláír, de nem küldi el az Apple-nek (gyors próba)
#
# A kapcsolók kombinálhatók: ./build.sh --local --install
#
# A darwin binárisokat Developer ID-vel aláírja és az Apple-lel hitelesítteti
# (notarizálás), ha megvannak a kulcsok az APPLE_SECRETS mappában:
#   developerid.pem  — Developer ID Application tanúsítvány + privát kulcs
#   notary-key.json  — App Store Connect API-kulcs (rcodesign encode-app-store-connect-api-key)
# Eszköz: rcodesign (github.com/indygreg/apple-platform-rs).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST="$ROOT/dist"
INSTALL_PATH="${VC_INSTALL_PATH:-$HOME/.local/bin/vc}"
APPLE_SECRETS="${APPLE_SECRETS:-$HOME/.secrets/apple}"
BUNDLE_ID="hu.feherkaroly.vc"

LOCAL=0
INSTALL=0
NOTARIZE=1
for arg in "$@"; do
  case "$arg" in
    --local) LOCAL=1 ;;
    --install) INSTALL=1 ;;
    --no-notarize) NOTARIZE=0 ;;
    --rollback)
      if [[ ! -f "$INSTALL_PATH.bak" ]]; then
        echo "Nincs mentés: $INSTALL_PATH.bak" >&2
        exit 1
      fi
      mv -f "$INSTALL_PATH.bak" "$INSTALL_PATH"
      echo "Visszaállítva: $("$INSTALL_PATH" --version)"
      exit 0
      ;;
    -h|--help) sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Ismeretlen kapcsoló: $arg (lásd: ./build.sh --help)" >&2; exit 1 ;;
  esac
done

VERSION="$(sed -n 's/^var Version = "\(.*\)"$/\1/p' "$ROOT/main.go")"
if [[ -z "$VERSION" ]]; then
  echo "Nem találom a verziót a main.go-ban (var Version = \"X.Y.Z\")" >&2
  exit 1
fi

if [[ $LOCAL -eq 1 ]]; then
  TARGETS=("linux/amd64")
else
  TARGETS=("darwin/arm64" "linux/amd64" "windows/amd64")
fi

mkdir -p "$DIST"
cd "$ROOT"
echo "vc $VERSION build"

for target in "${TARGETS[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  out="$DIST/vc-$goos-$goarch"
  [[ "$goos" == "windows" ]] && out="$out.exe"
  # CGO_ENABLED=0: statikus bináris, mint a korábbi release-ek
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags "-s -w -X main.Version=$VERSION" -o "$out" .
  printf '  %-26s %6s KB\n' "$(basename "$out")" "$(( $(stat -c %s "$out") / 1024 ))"
done

# macOS: aláírás + hitelesítés, különben a Gatekeeper minden új kiadásnál rákérdez
darwin_bins=("$DIST"/vc-darwin-*)
if [[ $LOCAL -eq 0 && -f "$APPLE_SECRETS/developerid.pem" ]]; then
  command -v rcodesign >/dev/null || { echo "rcodesign hiányzik" >&2; exit 1; }
  for bin in "${darwin_bins[@]}"; do
    rcodesign sign --pem-file "$APPLE_SECRETS/developerid.pem" \
      --code-signature-flags runtime --binary-identifier "$BUNDLE_ID" "$bin" >/dev/null 2>&1
    echo "  aláírva: $(basename "$bin")"
  done
  if [[ $NOTARIZE -eq 1 && -f "$APPLE_SECRETS/notary-key.json" ]]; then
    zipfile="$DIST/notarize-$VERSION.zip"
    rm -f "$zipfile"
    zip -q -j "$zipfile" "${darwin_bins[@]}"
    echo "  hitelesítés az Apple-nél..."
    log="$(rcodesign notary-submit --api-key-path "$APPLE_SECRETS/notary-key.json" --wait "$zipfile" 2>&1)" || {
      echo "$log" | tail -20 >&2
      echo "A hitelesítés nem sikerült" >&2
      exit 1
    }
    rm -f "$zipfile"
    echo "  hitelesítve: $(echo "$log" | grep -o '"status": "[A-Za-z]*"' | tail -1 | cut -d'"' -f4)"
  fi
elif [[ $LOCAL -eq 0 ]]; then
  echo "  (nincs $APPLE_SECRETS/developerid.pem — a darwin binárisok aláíratlanok)"
fi

if [[ $INSTALL -eq 1 ]]; then
  src="$DIST/vc-linux-amd64"
  if [[ ! -f "$src" ]]; then
    echo "Hiányzik: $src" >&2
    exit 1
  fi
  mkdir -p "$(dirname "$INSTALL_PATH")"
  [[ -f "$INSTALL_PATH" ]] && cp -p "$INSTALL_PATH" "$INSTALL_PATH.bak"
  # atomikus csere: egy futó vc példányt nem zavar
  cp "$src" "$INSTALL_PATH.new"
  chmod 755 "$INSTALL_PATH.new"
  mv -f "$INSTALL_PATH.new" "$INSTALL_PATH"
  echo "Telepítve: $INSTALL_PATH ($("$INSTALL_PATH" --version))"
fi
