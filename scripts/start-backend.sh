#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT/backend"
export APP_ADDR=${APP_ADDR:-:8080}
export APP_DB=${APP_DB:-$ROOT/license-saas.db}
export APP_JWT_SECRET=${APP_JWT_SECRET:-dev-secret-change-me}
export APP_ADMIN_USER=${APP_ADMIN_USER:-admin}
export APP_ADMIN_PASS=${APP_ADMIN_PASS:-admin123}
export APP_FRONTEND_DIST=${APP_FRONTEND_DIST:-$ROOT/frontend/dist}
go run ./cmd/server
