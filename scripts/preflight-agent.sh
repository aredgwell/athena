#!/usr/bin/env bash
set -euo pipefail

echo "== Athena Agent Preflight =="

echo "[1/2] Tooling check"
task ai:tools:check

echo "[2/2] AI memory check"
task ai:check

echo "Preflight OK"
