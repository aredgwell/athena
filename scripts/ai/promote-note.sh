#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "${SCRIPT_DIR}/common.sh"

RAW_PATH="${1:-}"
TARGET_DOC="${2:-}"

if [[ -z "${RAW_PATH}" || -z "${TARGET_DOC}" ]]; then
  echo "Usage: $0 <path> <target-doc>" >&2
  exit 2
fi

NOTE_PATH="$(resolve_note_path "${RAW_PATH}")"
TODAY="$(today_iso)"

frontmatter_set "${NOTE_PATH}" status "promoted"
frontmatter_set "${NOTE_PATH}" promotion_target "${TARGET_DOC}"
frontmatter_set "${NOTE_PATH}" updated "${TODAY}"

if rg -q '^## Promotion$' "${NOTE_PATH}"; then
  printf -- '- %s: Canonicalized into `%s`.\n' "${TODAY}" "${TARGET_DOC}" >>"${NOTE_PATH}"
else
  {
    echo
    echo "## Promotion"
    printf -- '- %s: Canonicalized into `%s`.\n' "${TODAY}" "${TARGET_DOC}"
  } >>"${NOTE_PATH}"
fi

echo "Promoted ${NOTE_PATH} -> ${TARGET_DOC}"
reindex_ai >/dev/null
