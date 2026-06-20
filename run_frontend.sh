#!/usr/bin/env bash
# Запуск frontend (Vite + React) на http://localhost:5173
set -e
ROOT="$(cd "$(dirname "$0")" && pwd)"
# Use locally-downloaded Node if a system Node isn't on PATH.
if ! command -v node >/dev/null 2>&1 && [ -d "$ROOT/.node/bin" ]; then
  export PATH="$ROOT/.node/bin:$PATH"
fi
cd "$ROOT/frontend"
if [ ! -d node_modules ]; then
  npm install
fi
exec npm run dev
