#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "${SCRIPT_DIR}/common.sh"

has_tag() {
  local file="$1"
  local tag="$2"

  awk -v tag="${tag}" '
    NR == 1 {
      if ($0 == "---") {
        in_fm = 1
        next
      }
      exit 1
    }

    in_fm && $0 == "---" {
      exit(found ? 0 : 1)
    }

    in_fm {
      if ($0 ~ /^tags:[[:space:]]*\[/) {
        line = $0
        sub(/^tags:[[:space:]]*\[/, "", line)
        sub(/\][[:space:]]*$/, "", line)
        gsub(/"/, "", line)
        n = split(line, parts, /,[[:space:]]*/)
        for (i = 1; i <= n; i++) {
          gsub(/^[[:space:]]+|[[:space:]]+$/, "", parts[i])
          if (parts[i] == tag) {
            found = 1
            exit 0
          }
        }
      }

      if ($0 ~ /^tags:[[:space:]]*$/) {
        in_tags = 1
        next
      }

      if (in_tags) {
        if ($0 ~ /^[[:space:]]*-[[:space:]]*/) {
          value = $0
          sub(/^[[:space:]]*-[[:space:]]*/, "", value)
          gsub(/^"/, "", value)
          gsub(/"$/, "", value)
          if (value == tag) {
            found = 1
            exit 0
          }
          next
        }
        in_tags = 0
      }
    }

    END {
      if (!found) {
        exit 1
      }
    }
  ' "${file}"
}

CANDIDATES=0

printf "Promotion candidates:\n"

for dir in context investigations troubleshooting wip improvements; do
  target_dir="${AI_DIR}/${dir}"
  [[ -d "${target_dir}" ]] || continue

  while IFS= read -r file; do
    status="$(frontmatter_get "${file}" status)"
    case "${status}" in
    promoted | closed | superseded)
      continue
      ;;
    esac

    target_doc="$(frontmatter_get "${file}" promotion_target)"
    reason=""

    if [[ -n "${target_doc}" ]]; then
      reason="has promotion_target"
    elif has_tag "${file}" "repeated-fix"; then
      reason="tagged repeated-fix"
      target_doc="-"
    else
      continue
    fi

    CANDIDATES=$((CANDIDATES + 1))
    printf "  - %s | status=%s | target=%s | %s\n" \
      "${file#${ROOT_DIR}/}" "${status:-unknown}" "${target_doc:-"-"}" "${reason}"
  done < <(find "${target_dir}" -type f -name '*.md' | sort)
done

if [[ "${CANDIDATES}" -eq 0 ]]; then
  echo "  (none)"
fi
