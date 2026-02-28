#!/usr/bin/env bash
# Hook: PreToolUse (Write|Edit) — 检查文件体积是否超标
# Go ≤ 500 行, Vue ≤ 400 行, JS ≤ 300 行
# engine.go 特殊：禁止新增函数

set -euo pipefail

INPUT=$(cat)
TOOL_NAME=$(printf '%s\n' "$INPUT" | jq -r '.tool_name // empty')
FILE_PATH=$(printf '%s\n' "$INPUT" | jq -r '.tool_input.file_path // empty')

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

EXT="${FILE_PATH##*.}"

# Determine effective content after tool applies
if [ "$TOOL_NAME" = "Write" ]; then
  CONTENT=$(printf '%s\n' "$INPUT" | jq -r '.tool_input.content // empty')
  if [ -z "$CONTENT" ]; then
    exit 0
  fi
elif [ "$TOOL_NAME" = "Edit" ]; then
  if [ ! -f "$FILE_PATH" ]; then
    exit 0
  fi
  OLD_STR=$(printf '%s\n' "$INPUT" | jq -r '.tool_input.old_string // empty')
  NEW_STR=$(printf '%s\n' "$INPUT" | jq -r '.tool_input.new_string // empty')
  if [ -z "$OLD_STR" ]; then
    exit 0
  fi
  # Simulate edit: replace old_string with new_string in file
  CONTENT=$(python3 -c "
import sys, json
with open(sys.argv[1], 'r') as f:
    content = f.read()
old = sys.argv[2]
new = sys.argv[3]
print(content.replace(old, new, 1))
" "$FILE_PATH" "$OLD_STR" "$NEW_STR" 2>/dev/null || cat "$FILE_PATH")
else
  exit 0
fi

LINE_COUNT=$(printf '%s\n' "$CONTENT" | wc -l | tr -d ' ')

case "$EXT" in
  go)
    if [ "$LINE_COUNT" -gt 500 ]; then
      echo "BLOCKED: Go 文件超 500 行 ($LINE_COUNT 行): $FILE_PATH" >&2
      exit 2
    fi
    # engine.go 特殊检查：禁止新增函数
    if [[ "$FILE_PATH" == *"engine.go" ]]; then
      if [ -f "$FILE_PATH" ]; then
        OLD_FUNC_COUNT=$(grep -c '^func ' "$FILE_PATH" 2>/dev/null || echo 0)
        NEW_FUNC_COUNT=$(printf '%s\n' "$CONTENT" | grep -c '^func ' 2>/dev/null || echo 0)
        if [ "$NEW_FUNC_COUNT" -gt "$OLD_FUNC_COUNT" ]; then
          echo "BLOCKED: engine.go 禁止新增函数 (原 $OLD_FUNC_COUNT 个, 新 $NEW_FUNC_COUNT 个)。新功能必须拆到独立文件" >&2
          exit 2
        fi
      fi
    fi
    ;;
  vue)
    if [ "$LINE_COUNT" -gt 400 ]; then
      echo "BLOCKED: Vue 组件超 400 行 ($LINE_COUNT 行): $FILE_PATH" >&2
      exit 2
    fi
    ;;
  js)
    if [ "$LINE_COUNT" -gt 300 ]; then
      echo "BLOCKED: JS 文件超 300 行 ($LINE_COUNT 行): $FILE_PATH" >&2
      exit 2
    fi
    ;;
esac

exit 0
