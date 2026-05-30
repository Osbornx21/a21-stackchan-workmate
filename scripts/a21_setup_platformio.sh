#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PYTHON_BIN="${PYTHON_BIN:-python3}"
A21_PLATFORMIO_VERSION="${A21_PLATFORMIO_VERSION:-6.1.19}"
A21_TOOLS_DIR="$ROOT/.a21-tools"
A21_PIO_VENV="$A21_TOOLS_DIR/platformio-venv"
A21_PIO_CORE="$A21_TOOLS_DIR/platformio-core"
A21_PIO="$A21_PIO_VENV/bin/pio"

case "$(printf '%s' "$ROOT" | tr '[:upper:]' '[:lower:]')" in
  *x21*|*v21*)
    echo "A21 firmware tool bootstrap refused: workspace path contains forbidden legacy identity" >&2
    exit 1
    ;;
esac

mkdir -p "$A21_TOOLS_DIR" "$A21_PIO_CORE"

if [ ! -x "$A21_PIO_VENV/bin/python" ]; then
  "$PYTHON_BIN" -m venv "$A21_PIO_VENV"
fi

current_version=""
if [ -x "$A21_PIO" ]; then
  current_version="$("$A21_PIO" --version 2>/dev/null | awk '/version/ {print $NF; exit}')"
fi

if [ "$current_version" != "$A21_PLATFORMIO_VERSION" ]; then
  "$A21_PIO_VENV/bin/python" -m pip install --upgrade pip
  "$A21_PIO_VENV/bin/python" -m pip install "platformio==$A21_PLATFORMIO_VERSION"
fi

PLATFORMIO_CORE_DIR="$A21_PIO_CORE" "$A21_PIO" --version
