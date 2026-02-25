#!/usr/bin/env bash
# Hook: PostToolUse (Write|Edit) — Go 架构边界检查
# engine ↛ agent, projection ↛ store, realtime ↛ engine

set -euo pipefail

INPUT=$(cat)
FILE_PATH=$(printf '%s\n' "$INPUT" | jq -r '.tool_input.file_path // empty')

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

# Only check .go files
if [[ "$FILE_PATH" != *.go ]]; then
  exit 0
fi

# Only check if file exists (post-tool, it should)
if [ ! -f "$FILE_PATH" ]; then
  exit 0
fi

# Extract import block
IMPORTS=$(sed -n '/^import/,/^)/p' "$FILE_PATH" 2>/dev/null || true)

# engine → agent boundary
if [[ "$FILE_PATH" == *"internal/engine/"* ]]; then
  if echo "$IMPORTS" | grep -q 'internal/agent'; then
    echo "BLOCKED: engine 包禁止 import agent — 状态机不能知道 AI 的存在" >&2
    exit 2
  fi
fi

# projection → store boundary
if [[ "$FILE_PATH" == *"internal/projection/"* ]]; then
  if echo "$IMPORTS" | grep -q 'internal/store'; then
    echo "BLOCKED: projection 包禁止 import store — 只做读过滤不写存储" >&2
    exit 2
  fi
fi

# realtime → engine boundary
if [[ "$FILE_PATH" == *"internal/realtime/"* ]]; then
  if echo "$IMPORTS" | grep -q 'internal/engine'; then
    echo "BLOCKED: realtime 包禁止 import engine — WebSocket 层不含游戏逻辑" >&2
    exit 2
  fi
fi

exit 0
