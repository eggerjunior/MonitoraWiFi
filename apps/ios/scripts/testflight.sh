#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ ! -f scripts/asc.env ]]; then
  echo "ERRO: scripts/asc.env nao encontrado." >&2
  echo "      Copie scripts/asc.env.example para scripts/asc.env e preencha." >&2
  exit 1
fi

# shellcheck disable=SC1091
source scripts/asc.env

: "${ASC_KEY_ID:?defina ASC_KEY_ID em scripts/asc.env}"
: "${ASC_ISSUER_ID:?defina ASC_ISSUER_ID em scripts/asc.env}"
: "${ASC_KEY_PATH:?defina ASC_KEY_PATH em scripts/asc.env}"

[[ -f "$ASC_KEY_PATH" ]] || { echo "ERRO: chave .p8 nao encontrada em: $ASC_KEY_PATH" >&2; exit 1; }

SCHEME="${SCHEME:-EggerNetworkIntelligence}"
PROJECT="${PROJECT:-EggerNetworkIntelligence.xcodeproj}"

echo "==> xcodegen generate"
xcodegen generate >/dev/null

SETTINGS="$(xcodebuild -project "$PROJECT" -scheme "$SCHEME" -showBuildSettings 2>/dev/null)"
VERSION="$(printf '%s\n' "$SETTINGS" | awk -F' = ' '/ MARKETING_VERSION = /{print $2; exit}')"
BUILD="$(printf '%s\n' "$SETTINGS" | awk -F' = ' '/ CURRENT_PROJECT_VERSION = /{print $2; exit}')"
GIT_COMMIT="$(git rev-parse --short=8 HEAD 2>/dev/null || echo dev)"

if [[ -f Sources/Version/VersionHistory.swift ]]; then
  CURRENT_HISTORY="$(awk '
    /VersionEntry\(/ { in_entry=1; version=""; build="" }
    in_entry && /version:/ {
      if (match($0, /version: "([^"]+)"/)) version=substr($0, RSTART + 10, RLENGTH - 11)
    }
    in_entry && /build:/ {
      if (match($0, /build: "([^"]+)"/)) build=substr($0, RSTART + 8, RLENGTH - 9)
    }
    in_entry && /isCurrent: true/ {
      print version "|" build
      exit
    }
  ' Sources/Version/VersionHistory.swift)"

  if [[ "$CURRENT_HISTORY" != "${VERSION}|${BUILD}" ]]; then
    echo "ERRO: VersionHistory atual (${CURRENT_HISTORY:-nao encontrado}) nao confere com project.yml (${VERSION}|${BUILD})." >&2
    exit 1
  fi
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo "ERRO: ha mudancas nao commitadas. Faca commit + push antes do build de distribuicao." >&2
  git status --short >&2
  exit 1
fi

echo "==> Enviando ${SCHEME} ${VERSION} (${BUILD}) - commit ${GIT_COMMIT}"

ARCH_DIR="$HOME/Library/Developer/Xcode/Archives/$(date +%Y-%m-%d)"
mkdir -p "$ARCH_DIR"
ARCH_PATH="${ARCH_DIR}/${SCHEME} ${VERSION} (${BUILD}).xcarchive"
EXPORT_DIR="$(mktemp -d)"

echo "==> Arquivando (Release)"
# -authenticationKey* precisa estar aqui tambem, nao so no -exportArchive:
# com CODE_SIGN_STYLE Automatic e nenhuma conta Apple logada no Xcode local,
# o archive falha com "No Accounts" sem a chave da App Store Connect ja
# neste passo.
xcodebuild -project "$PROJECT" -scheme "$SCHEME" \
  -configuration Release \
  -destination 'generic/platform=iOS' \
  -archivePath "$ARCH_PATH" \
  -allowProvisioningUpdates \
  -authenticationKeyPath "$ASC_KEY_PATH" \
  -authenticationKeyID "$ASC_KEY_ID" \
  -authenticationKeyIssuerID "$ASC_ISSUER_ID" \
  GIT_COMMIT="$GIT_COMMIT" \
  archive

echo "==> Exportando e enviando ao App Store Connect"
xcodebuild -exportArchive \
  -archivePath "$ARCH_PATH" \
  -exportOptionsPlist scripts/ExportOptions.plist \
  -exportPath "$EXPORT_DIR" \
  -allowProvisioningUpdates \
  -authenticationKeyPath "$ASC_KEY_PATH" \
  -authenticationKeyID "$ASC_KEY_ID" \
  -authenticationKeyIssuerID "$ASC_ISSUER_ID"

echo ""
echo "==> OK: ${VERSION} (${BUILD}) enviado ao TestFlight."
echo "    Acompanhe em App Store Connect > TestFlight."
