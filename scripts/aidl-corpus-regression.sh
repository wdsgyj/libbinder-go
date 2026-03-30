#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/aidl-corpus-regression.sh

Runs host-side AIDL corpus regression:
  1. no-package multi-file compile smoke
  2. AOSP binder corpus generate + compile
EOF
}

main() {
  if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
  fi

  (
    cd "${ROOT_DIR}"
    go test ./cmd/aidlgen -run 'TestRunGo(NoPackageDirectoryCompiles|AOSPBinderCorpus)$' -count=1
  )
}

main "$@"
