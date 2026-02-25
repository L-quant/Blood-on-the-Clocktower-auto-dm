#!/usr/bin/env bash
# Hook: PostToolUse (Write|Edit) — Vue 架构边界检查
# 组件 ↛ ApiService, modules ↛ modules

set -euo pipefail

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

# Only check if file exists
if [ ! -f "$FILE_PATH" ]; then
  exit 0
fi

# Vue components must not import ApiService directly
if [[ "$FILE_PATH" == *.vue ]]; then
  if grep -qE "import.*ApiService|from.*services/Api" "$FILE_PATH" 2>/dev/null; then
    echo "BLOCKED: 组件禁止直接 import ApiService，通过 Vuex action 调用" >&2
    exit 2
  fi
fi

# Vuex modules must not import each other directly
if [[ "$FILE_PATH" == *"store/modules/"*.js ]]; then
  if grep -qE "from.*modules/" "$FILE_PATH" 2>/dev/null; then
    echo "BLOCKED: Vuex modules 之间禁止直接 import，用 rootGetters" >&2
    exit 2
  fi
fi

exit 0
