#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
AI_DIR="${ROOT_DIR}/.ai"

today_iso() {
  date +"%Y-%m-%d"
}

resolve_note_path() {
  local raw_path="$1"
  local candidate=""

  if [[ -f "${raw_path}" ]]; then
    candidate="${raw_path}"
  elif [[ -f "${ROOT_DIR}/${raw_path}" ]]; then
    candidate="${ROOT_DIR}/${raw_path}"
  elif [[ -f "${AI_DIR}/${raw_path}" ]]; then
    candidate="${AI_DIR}/${raw_path}"
  else
    echo "ERROR: Note file not found: ${raw_path}" >&2
    return 1
  fi

  local abs_dir
  abs_dir="$(cd -- "$(dirname -- "${candidate}")" && pwd)"
  local abs_path="${abs_dir}/$(basename -- "${candidate}")"

  case "${abs_path}" in
    "${AI_DIR}"/*)
      printf "%s\n" "${abs_path}"
      ;;
    *)
      echo "ERROR: Refusing to modify file outside .ai/: ${abs_path}" >&2
      return 1
      ;;
  esac
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

frontmatter_set() {
  local file="$1"
  local key="$2"
  local value="$3"
  local tmp

  tmp="$(mktemp)"

  awk -v key="${key}" -v value="${value}" '
    NR == 1 {
      if ($0 == "---") {
        in_fm = 1
        print
        next
      }

      print "---"
      print key ": \"" value "\""
      print "---"
      print
      next
    }

    in_fm && $0 == "---" {
      if (!seen_key) {
        print key ": \"" value "\""
      }
      print
      in_fm = 0
      next
    }

    in_fm {
      pattern = "^" key ":[[:space:]]*"
      if ($0 ~ pattern) {
        print key ": \"" value "\""
        seen_key = 1
        next
      }
    }

    {
      print
    }

    END {
      if (NR == 0) {
        print "---"
        print key ": \"" value "\""
        print "---"
      } else if (in_fm) {
        if (!seen_key) {
          print key ": \"" value "\""
        }
        print "---"
      }
    }
  ' "${file}" >"${tmp}"

  mv "${tmp}" "${file}"
}

reindex_ai() {
  "${ROOT_DIR}/scripts/ai/reindex.sh"
}
