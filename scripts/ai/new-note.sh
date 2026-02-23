#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "${SCRIPT_DIR}/common.sh"

TYPE="${1:-}"
SLUG="${2:-}"
TITLE="${3:-}"

if [[ -z "${TYPE}" || -z "${SLUG}" ]]; then
  echo "Usage: $0 <type> <slug> [title]" >&2
  echo "Types: context | investigation | troubleshooting | wip | improvement" >&2
  exit 2
fi

case "${TYPE}" in
context)
  NOTE_DIR="context"
  NOTE_TYPE="context"
  ;;
investigation | investigations)
  NOTE_DIR="investigations"
  NOTE_TYPE="investigation"
  ;;
troubleshooting)
  NOTE_DIR="troubleshooting"
  NOTE_TYPE="troubleshooting"
  ;;
wip)
  NOTE_DIR="wip"
  NOTE_TYPE="wip"
  ;;
improvement | improvements)
  NOTE_DIR="improvements"
  NOTE_TYPE="improvement"
  ;;
*)
  echo "ERROR: Unknown note type '${TYPE}'" >&2
  exit 2
  ;;
esac

if [[ -z "${TITLE}" ]]; then
  TITLE="$(printf '%s' "${SLUG}" | tr '-' ' ')"
fi

DATE_ISO="$(today_iso)"
DATE_TAG="$(date +"%Y%m%d")"
NOTE_PATH="${AI_DIR}/${NOTE_DIR}/${DATE_ISO}-${SLUG}.md"
NOTE_ID="${NOTE_TYPE}-${DATE_TAG}-${SLUG}"

if [[ -e "${NOTE_PATH}" ]]; then
  echo "ERROR: Note already exists: ${NOTE_PATH}" >&2
  exit 1
fi

cat >"${NOTE_PATH}" <<EONOTE
---
id: "${NOTE_ID}"
title: "${TITLE}"
type: "${NOTE_TYPE}"
status: "active"
created: "${DATE_ISO}"
updated: "${DATE_ISO}"
related: []
promotion_target: ""
supersedes: []
tags: []
---

# ${TITLE}

## Summary

TBD.

## Facts

- TBD

## Decisions

- TBD

## Next Steps

1. TBD

## Promotion Candidate

Target canonical doc (if any): \`docs/...\`
EONOTE

echo "Created ${NOTE_PATH}"
reindex_ai >/dev/null
