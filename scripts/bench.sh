#!/usr/bin/env bash
# bench.sh — run all benchmarks and record results with timestamp.
#
# Usage:
#   ./scripts/bench.sh              # run and record
#   ./scripts/bench.sh --count 3    # run 3 times for stable averages
#
# Results are written to:
#   testdata/compat/bench-results/YYYY-MM-DDTHH-MM-SS.txt  (timestamped)
#   testdata/compat/bench-results/latest.txt                (overwritten each run)
#
# To compare two runs:
#   go install golang.org/x/perf/cmd/benchstat@latest
#   benchstat testdata/compat/bench-results/<old>.txt testdata/compat/bench-results/latest.txt

set -euo pipefail

COUNT=${2:-1}
if [[ "${1:-}" == "--count" ]]; then
  COUNT=$2
fi

RESULTS_DIR="testdata/compat/bench-results"
mkdir -p "$RESULTS_DIR"

TIMESTAMP=$(date +"%Y-%m-%dT%H-%M-%S")
OUT="$RESULTS_DIR/${TIMESTAMP}.txt"

echo "Running benchmarks (count=${COUNT})..."
echo "# goos: $(go env GOOS)" > "$OUT"
echo "# goarch: $(go env GOARCH)" >> "$OUT"
echo "# pkg: github.com/pinealctx/gcode/testdata/compat" >> "$OUT"
echo "# timestamp: ${TIMESTAMP}" >> "$OUT"
echo "# go: $(go version)" >> "$OUT"
echo "" >> "$OUT"

go test -run=^$ -bench=. -benchmem -count="${COUNT}" \
  github.com/pinealctx/gcode/testdata/compat \
  | tee -a "$OUT"

cp "$OUT" "$RESULTS_DIR/latest.txt"

echo ""
echo "Results saved to: $OUT"
echo "Latest:           $RESULTS_DIR/latest.txt"
echo ""
echo "To compare with a previous run:"
echo "  benchstat $RESULTS_DIR/<previous>.txt $RESULTS_DIR/latest.txt"
