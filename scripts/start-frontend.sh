#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT/frontend"
export VITE_API_BASE=${VITE_API_BASE:-http://127.0.0.1:8080}
npm run dev -- --port 5173
