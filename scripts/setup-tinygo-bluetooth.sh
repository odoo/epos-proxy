#!/usr/bin/env bash
# scripts/setup-tinygo-bluetooth.sh
#
# Clones tinygo.org/x/bluetooth v0.15.0 into ./tinygo-bluetooth/ and applies
# all patch files from ./tinygo-bluetooth-patches/.
#
# Usage:
#   ./scripts/setup-tinygo-bluetooth.sh            # skip if already present
#   ./scripts/setup-tinygo-bluetooth.sh --force    # re-clone and re-apply

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_DIR="${REPO_ROOT}/tinygo-bluetooth"
PATCHES_DIR="${REPO_ROOT}/tinygo-bluetooth-patches"
UPSTREAM_URL="https://github.com/tinygo-org/bluetooth.git"
UPSTREAM_TAG="v0.15.0"

FORCE=false
if [[ "${1:-}" == "--force" ]]; then
  FORCE=true
fi

# ── Clone upstream if needed ──────────────────────────────────────────────────
if [[ -f "${TARGET_DIR}/go.mod" ]] && [[ "${FORCE}" == "false" ]]; then
  echo "✓ tinygo-bluetooth already present (use --force to re-clone)"
else
  echo "▶ Cloning tinygo-org/bluetooth ${UPSTREAM_TAG}..."

  # Remove existing content so we get a clean clone
  rm -rf "${TARGET_DIR}"
  mkdir -p "${TARGET_DIR}"

  git clone \
    --depth 1 \
    --branch "${UPSTREAM_TAG}" \
    "${UPSTREAM_URL}" \
    "${TARGET_DIR}" \
    --quiet

  # Remove the upstream .git — this is just a plain source directory
  rm -rf "${TARGET_DIR}/.git"

  echo "  ✓ Cloned ${UPSTREAM_TAG}"
fi

# ── Apply patches ─────────────────────────────────────────────────────────────
if [[ ! -d "${PATCHES_DIR}" ]]; then
  echo "⚠ No patches directory found at ${PATCHES_DIR}, skipping"
  exit 0
fi

PATCH_FILES=("${PATCHES_DIR}"/*.patch)
if [[ ${#PATCH_FILES[@]} -eq 0 ]] || [[ ! -f "${PATCH_FILES[0]}" ]]; then
  echo "⚠ No .patch files found in ${PATCHES_DIR}"
  exit 0
fi

echo "▶ Applying ${#PATCH_FILES[@]} patch(es)..."
for patch_file in "${PATCH_FILES[@]}"; do
  echo "  ✎ $(basename "${patch_file}")"

  if patch \
      --directory="${TARGET_DIR}" \
      --strip=1 \
      --forward \
      --dry-run \
      < "${patch_file}" >/dev/null 2>&1; then

    patch \
      --directory="${TARGET_DIR}" \
      --strip=1 \
      --forward \
      < "${patch_file}"

  else
    echo "    ✓ Already applied (or cannot be applied), skipping"
  fi
done

echo "✅ tinygo-bluetooth is ready (${UPSTREAM_TAG} + patches)"
