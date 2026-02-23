#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
AI_DIR="${ROOT_DIR}/.ai"
INDEX_FILE="${AI_DIR}/index.yaml"
NOW_UTC="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

yaml_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  printf '%s' "${value}"
}

frontmatter_get() {
  local file="$1"
  local key="$2"

  awk -v key="${key}" '
    NR == 1 {
      if ($0 == "---") {
        in_fm = 1
        next
      }
      exit
    }

    in_fm && $0 == "---" {
      exit
    }

    in_fm {
      pattern = "^" key ":[[:space:]]*"
      if ($0 ~ pattern) {
        sub(pattern, "", $0)
        gsub(/^"/, "", $0)
        gsub(/"$/, "", $0)
        print $0
        exit
      }
    }
  ' "${file}"
}

infer_type_from_path() {
  local rel_path="$1"

  case "${rel_path}" in
    .ai/context/*)
      printf 'context'
      ;;
    .ai/investigations/*)
      printf 'investigation'
      ;;
    .ai/troubleshooting/*)
      printf 'troubleshooting'
      ;;
    .ai/wip/*)
      printf 'wip'
      ;;
    .ai/improvements/*)
      printf 'improvement'
      ;;
    .ai/memory/*)
      printf 'memory'
      ;;
    .ai/templates/*)
      printf 'template'
      ;;
    *)
      printf 'unknown'
      ;;
  esac
}

heading_title() {
  local file="$1"
  awk '/^# / { sub(/^# /, "", $0); print; exit }' "${file}"
}

collection_count() {
  local name="$1"
  local dir="${AI_DIR}/${name}"

  if [[ ! -d "${dir}" ]]; then
    printf '0'
    return
  fi

  find "${dir}" -type f -name '*.md' | wc -l | tr -d ' '
}

tmp_file="$(mktemp)"

{
  echo "version: 1"
  echo "generated_at: \"${NOW_UTC}\""
  echo "root: \".ai\""
  echo "collections:"
  echo "  context: $(collection_count context)"
  echo "  investigations: $(collection_count investigations)"
  echo "  troubleshooting: $(collection_count troubleshooting)"
  echo "  wip: $(collection_count wip)"
  echo "  improvements: $(collection_count improvements)"
  echo "  memory: $(collection_count memory)"
  echo "  templates: $(collection_count templates)"
  echo "entries:"

  while IFS= read -r file; do
    rel_path="${file#${ROOT_DIR}/}"

    type="$(frontmatter_get "${file}" type)"
    if [[ -z "${type}" ]]; then
      type="$(infer_type_from_path "${rel_path}")"
    fi

    status="$(frontmatter_get "${file}" status)"
    if [[ -z "${status}" ]]; then
      status="legacy"
    fi

    updated="$(frontmatter_get "${file}" updated)"
    if [[ -z "${updated}" ]]; then
      updated="$(date -r "${file}" +"%Y-%m-%d")"
    fi

    title="$(frontmatter_get "${file}" title)"
    if [[ -z "${title}" ]]; then
      title="$(heading_title "${file}")"
    fi
    if [[ -z "${title}" ]]; then
      title="$(basename -- "${file}")"
    fi

    echo "  - path: \"$(yaml_escape "${rel_path}")\""
    echo "    type: \"$(yaml_escape "${type}")\""
    echo "    status: \"$(yaml_escape "${status}")\""
    echo "    updated: \"$(yaml_escape "${updated}")\""
    echo "    title: \"$(yaml_escape "${title}")\""
  done < <(find "${AI_DIR}" -type f -name '*.md' ! -path "${AI_DIR}/templates/*" | sort)
} >"${tmp_file}"

mv "${tmp_file}" "${INDEX_FILE}"
echo "Updated ${INDEX_FILE}"
