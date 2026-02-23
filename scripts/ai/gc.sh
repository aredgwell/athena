#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "${SCRIPT_DIR}/common.sh"

DAYS="${1:-45}"
if ! [[ "${DAYS}" =~ ^[0-9]+$ ]]; then
  echo "ERROR: DAYS must be an integer, got '${DAYS}'" >&2
  exit 2
fi

TODAY="$(today_iso)"
CHANGED=0

for dir in context troubleshooting wip; do
  target_dir="${AI_DIR}/${dir}"
  [[ -d "${target_dir}" ]] || continue

  while IFS= read -r file; do
    status="$(frontmatter_get "${file}" status)"
    if [[ "${status}" == "active" ]]; then
      frontmatter_set "${file}" status "stale"
      frontmatter_set "${file}" updated "${TODAY}"
      echo "Marked stale: ${file#${ROOT_DIR}/}"
      CHANGED=$((CHANGED + 1))
    fi
  done < <(find "${target_dir}" -type f -name '*.md' -mtime +"${DAYS}" | sort)
done

if [[ "${CHANGED}" -gt 0 ]]; then
  reindex_ai >/dev/null
fi

echo "GC complete. ${CHANGED} note(s) marked stale."
