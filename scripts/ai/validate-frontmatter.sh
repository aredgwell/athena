#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "${SCRIPT_DIR}/common.sh"

FAILURES=0
WARNINGS=0
REQUIRED_KEYS=(id title type status created updated)

check_required_section() {
  local file="$1"
  local section="$2"

  if ! grep -Eq "^## ${section}$" "${file}"; then
    echo "FAIL: missing section '## ${section}' in ${file#${ROOT_DIR}/}"
    FAILURES=$((FAILURES + 1))
  fi
}

check_file() {
  local file="$1"

  if [[ "$(head -n1 "${file}" 2>/dev/null || true)" != "---" ]]; then
    echo "FAIL: missing frontmatter in ${file#${ROOT_DIR}/}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  for key in "${REQUIRED_KEYS[@]}"; do
    local value
    value="$(frontmatter_get "${file}" "${key}")"
    if [[ -z "${value}" ]]; then
      echo "FAIL: missing '${key}' in ${file#${ROOT_DIR}/}"
      FAILURES=$((FAILURES + 1))
    fi
  done

  local created
  local updated
  created="$(frontmatter_get "${file}" created)"
  updated="$(frontmatter_get "${file}" updated)"
  if [[ ! "${created}" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
    echo "FAIL: invalid 'created' date format in ${file#${ROOT_DIR}/}: '${created}'"
    FAILURES=$((FAILURES + 1))
  fi
  if [[ ! "${updated}" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
    echo "FAIL: invalid 'updated' date format in ${file#${ROOT_DIR}/}: '${updated}'"
    FAILURES=$((FAILURES + 1))
  fi
}

for dir in context investigations troubleshooting wip improvements memory; do
  target_dir="${AI_DIR}/${dir}"
  [[ -d "${target_dir}" ]] || continue

  while IFS= read -r file; do
    check_file "${file}"
  done < <(find "${target_dir}" -type f -name '*.md' | sort)
done

check_file "${AI_DIR}/README.md"

SESSION_NOTE="${AI_DIR}/context/session_state.md"
PLAN_NOTE="${AI_DIR}/memory/plan.md"

if [[ -f "${SESSION_NOTE}" ]]; then
  check_required_section "${SESSION_NOTE}" "Goal"
  check_required_section "${SESSION_NOTE}" "Working Set"
  check_required_section "${SESSION_NOTE}" "Next Actions"
else
  echo "FAIL: missing required session note: ${SESSION_NOTE#${ROOT_DIR}/}"
  FAILURES=$((FAILURES + 1))
fi

if [[ -f "${PLAN_NOTE}" ]]; then
  check_required_section "${PLAN_NOTE}" "Scope"
  check_required_section "${PLAN_NOTE}" "Files to touch"
  check_required_section "${PLAN_NOTE}" "Validation plan"
else
  echo "FAIL: missing required plan note: ${PLAN_NOTE#${ROOT_DIR}/}"
  FAILURES=$((FAILURES + 1))
fi

if [[ "${FAILURES}" -gt 0 ]]; then
  echo "Frontmatter validation failed: ${FAILURES} error(s), ${WARNINGS} warning(s)." >&2
  exit 1
fi

echo "Frontmatter validation passed with ${WARNINGS} warning(s)."
