#!/bin/sh
set -e

fail=0

big=$(find internal cmd -name '*.go' ! -name '*_test.go' -print0 | xargs -0 wc -l | awk '$1>800 && $2!="total"{print}')
if [ -n "$big" ]; then
  echo "RED product files >800 lines:"
  echo "$big"
  fail=1
fi

bigt=$(find internal cmd -name '*_test.go' -print0 | xargs -0 wc -l | awk '$1>1000 && $2!="total"{print}')
if [ -n "$bigt" ]; then
  echo "RED test files >1000 lines:"
  echo "$bigt"
  fail=1
fi

package_leak=$(go list -deps ./cmd/a21 2>/dev/null | grep -iE 'bench|demo|evidence|professional' || true)
if [ -n "$package_leak" ]; then
  echo "RED product binary package path contains lab scaffold:"
  echo "$package_leak"
  fail=1
fi

tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT
go build -o "$tmp_dir/a21" ./cmd/a21
symbol_leak=$(go tool nm "$tmp_dir/a21" | grep -E 'run(Demo|ProviderLatencyBench|XiaozhiVoiceBench|XiaozhiProfessionalBench|XiaozhiPhysicalEvidence|XiaozhiPhysicalPRDReview|XiaozhiInstrumentObservation|PhysicalStackChanEvidence|LatencyBench)|RunLab|handleSimulator|simulatorHTML' || true)
if [ -n "$symbol_leak" ]; then
  echo "RED product binary links lab/simulator symbols:"
  echo "$symbol_leak"
  fail=1
fi

nb=$(git branch | wc -l | tr -d ' ')
if [ "$nb" -gt 3 ]; then
  echo "RED active branches >3: $nb"
  fail=1
fi

go build ./...
go vet ./...

exit "$fail"
