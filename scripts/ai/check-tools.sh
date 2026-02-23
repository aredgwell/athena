#!/usr/bin/env bash
set -euo pipefail

STRICT=0
OUTPUT="${1:-}"

if [[ "${OUTPUT}" == "--strict" ]]; then
  STRICT=1
fi

print_group() {
  local group="$1"
  local required_flag="$2"
  shift 2
  local tools=("$@")
  local missing=0

  echo
  echo "[${group}]"
  printf "%-20s %-10s %s\n" "tool" "status" "path"
  printf "%-20s %-10s %s\n" "----" "------" "----"

  for tool in "${tools[@]}"; do
    if path="$(command -v "${tool}" 2>/dev/null)"; then
      printf "%-20s %-10s %s\n" "${tool}" "present" "${path}"
    else
      printf "%-20s %-10s %s\n" "${tool}" "missing" "-"
      missing=$((missing + 1))
    fi
  done

  if [[ "${required_flag}" -eq 1 ]]; then
    MISSING_REQUIRED="${missing}"
  fi
}

REQUIRED_TOOLS=(
  git
  rg
  jq
  yq
  task
)

RECOMMENDED_TOOLS=(
  difft
  htmlq
  repomix
  fzf
  tree
  shfmt
  shellcheck
  uv
)

INFRA_TOOLS=(
  terraform
  tflint
  ansible
  ansible-lint
  packer
  sops
)

PLATFORM_TOOLS=(
  gh
  aws
  kubectl
  kustomize
  vault
)

MISSING_REQUIRED=0

print_group "required" 1 "${REQUIRED_TOOLS[@]}"
print_group "recommended" 0 "${RECOMMENDED_TOOLS[@]}"
print_group "infrastructure" 0 "${INFRA_TOOLS[@]}"
print_group "platform" 0 "${PLATFORM_TOOLS[@]}"

echo
if [[ "${MISSING_REQUIRED}" -eq 0 ]]; then
  echo "Required tooling status: OK"
else
  echo "Required tooling status: ${MISSING_REQUIRED} missing"
fi

if [[ "${STRICT}" -eq 1 && "${MISSING_REQUIRED}" -gt 0 ]]; then
  exit 1
fi
