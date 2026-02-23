#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "${SCRIPT_DIR}/common.sh"

RAW_PATH="${1:-}"
STATUS="${2:-closed}"

if [[ -z "${RAW_PATH}" ]]; then
  echo "Usage: $0 <path> [status]" >&2
  exit 2
fi

case "${STATUS}" in
  closed | stale | superseded | promoted)
    ;;
  *)
    echo "ERROR: Unsupported status '${STATUS}'" >&2
    echo "Allowed: closed | stale | superseded | promoted" >&2
    exit 2
    ;;
esac

NOTE_PATH="$(resolve_note_path "${RAW_PATH}")"
TODAY="$(today_iso)"

frontmatter_set "${NOTE_PATH}" status "${STATUS}"
frontmatter_set "${NOTE_PATH}" updated "${TODAY}"

echo "Updated ${NOTE_PATH} -> status=${STATUS}"
reindex_ai >/dev/null
