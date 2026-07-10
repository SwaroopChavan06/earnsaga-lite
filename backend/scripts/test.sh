#!/usr/bin/env bash
# Run unit tests with readable per-case output.
# Format:  -: TestName   then   -> PASS | -> FAIL  << FAILED
# Skips packages that have no *_test.go.
set -uo pipefail

RED=$'\033[31m'
GREEN=$'\033[32m'
YELLOW=$'\033[33m'
GRAY=$'\033[90m'
RESET=$'\033[0m'

mapfile -t packages < <(go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./... | sed '/^$/d')

if [[ ${#packages[@]} -eq 0 ]]; then
  echo "No packages with tests found."
  exit 1
fi

echo "Testing ${#packages[@]} package(s)"
echo

pass_count=0
fail_count=0
failed=()

while IFS= read -r line || [[ -n "$line" ]]; do
  if [[ "$line" =~ ^===[[:space:]]RUN[[:space:]]+([^[:space:]]+) ]]; then
    echo "-: ${BASH_REMATCH[1]}"
    continue
  fi

  if [[ "$line" =~ ^[[:space:]]*---[[:space:]](PASS|FAIL|SKIP):[[:space:]]+([^[:space:]]+)[[:space:]]+\(([^)]+)\) ]]; then
    status="${BASH_REMATCH[1]}"
    name="${BASH_REMATCH[2]}"
    dur="${BASH_REMATCH[3]}"
    if [[ "$status" == "PASS" ]]; then
      echo "${GREEN}-> PASS  ${name}  (${dur})${RESET}"
      ((pass_count++)) || true
    elif [[ "$status" == "SKIP" ]]; then
      echo "${YELLOW}-> SKIP  ${name}  (${dur})${RESET}"
    else
      echo "${RED}-> FAIL  ${name}  (${dur})  << FAILED${RESET}"
      ((fail_count++)) || true
      failed+=("$name")
    fi
    continue
  fi

  if [[ "$line" =~ ^ok[[:space:]]+([^[:space:]]+) ]]; then
    echo
    echo "${GREEN}ok  ${BASH_REMATCH[1]}${RESET}"
    echo
    continue
  fi

  if [[ "$line" =~ ^FAIL[[:space:]]+([^[:space:]]+) ]]; then
    echo
    echo "${RED}FAIL  ${BASH_REMATCH[1]}  << PACKAGE FAILED${RESET}"
    echo
    continue
  fi

  if [[ "$line" =~ (Error[[:space:]]Trace|Error:|Messages:|expected|got[[:space:]]|panic:|_test\.go:) ]]; then
    echo "${RED}${line}${RESET}"
    continue
  fi

  if [[ "$line" == "PASS" || "$line" == "FAIL" ]]; then
    continue
  fi

  if [[ "$line" =~ ^#[[:space:]] ]]; then
    echo "${GRAY}${line}${RESET}"
  fi
done < <(go test -v -count=1 "${packages[@]}" 2>&1)
code=${PIPESTATUS[0]}

echo "----------------------------------------"
echo "Summary: $pass_count passed, $fail_count failed"
if (( ${#failed[@]} > 0 )); then
  echo
  echo "${RED}FAILED TESTS:${RESET}"
  for f in "${failed[@]}"; do
    echo "${RED}  << $f${RESET}"
  done
fi
echo "----------------------------------------"

exit "$code"
