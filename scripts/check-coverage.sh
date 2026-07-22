#!/usr/bin/env sh
set -eu
PROFILE=${1:-coverage.out}
MINIMUM=${COVERAGE_MINIMUM:-80}
TOTAL=$(go tool cover -func="$PROFILE" | awk '/^total:/ {gsub("%", "", $3); print $3}')
awk -v total="$TOTAL" -v minimum="$MINIMUM" 'BEGIN {
  if (total + 0 < minimum + 0) {
    printf "coverage %.1f%% is below required %.1f%%\n", total, minimum > "/dev/stderr"
    exit 1
  }
  printf "coverage %.1f%% meets required %.1f%%\n", total, minimum
}'
