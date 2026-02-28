#!/usr/bin/env bash
# Hook: PostToolUse (Write|Edit) — 自动同步全局 plans 到项目 plans 目录
# 检测写入 ~/.claude/plans/ 的文件，自动复制到 $CLAUDE_PROJECT_DIR/.claude/plans/
# 命名规则：YYYY-MM-DD-NNN-<从标题提取的功能名>.md

set -uo pipefail

INPUT=$(cat)
FILE_PATH=$(printf '%s\n' "$INPUT" | jq -r '.tool_input.file_path // empty')

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

# 只处理全局 ~/.claude/plans/ 下的文件
GLOBAL_PLANS_DIR="$HOME/.claude/plans"
if [[ "$FILE_PATH" != "$GLOBAL_PLANS_DIR/"* ]]; then
  exit 0
fi

# 确保文件存在
if [ ! -f "$FILE_PATH" ]; then
  exit 0
fi

PROJECT_PLANS_DIR="${CLAUDE_PROJECT_DIR:-.}/.claude/plans"
mkdir -p "$PROJECT_PLANS_DIR"

BASENAME=$(basename "$FILE_PATH")
MAPPING_FILE="$PROJECT_PLANS_DIR/.plan-mapping"
touch "$MAPPING_FILE"

# 查找是否已有此全局文件的映射
EXISTING_TARGET=$(grep "^${BASENAME}=" "$MAPPING_FILE" 2>/dev/null | head -1 | cut -d= -f2- || true)

if [ -n "$EXISTING_TARGET" ] && [ -f "$PROJECT_PLANS_DIR/$EXISTING_TARGET" ]; then
  cp "$FILE_PATH" "$PROJECT_PLANS_DIR/$EXISTING_TARGET"
  exit 0
fi

# 用 python3 生成规范文件名（正确处理 UTF-8）
TARGET_NAME=$(python3 -c "
import re, os, sys, glob
from datetime import date

plan_file = sys.argv[1]
plans_dir = sys.argv[2]
today = date.today().strftime('%Y-%m-%d')

# 从第一个 # 标题提取功能名
title = 'plan'
with open(plan_file, 'r', encoding='utf-8') as f:
    for line in f:
        if line.startswith('# '):
            title = line[2:].strip()
            break

# 生成 slug：优先取 em dash 后的中文部分，否则用完整标题
parts = re.split(r'[—–]+', title, maxsplit=1)
slug_src = parts[-1].strip() if len(parts) > 1 and parts[-1].strip() else title

# 清理：保留中文/英文/数字/空格，转连字符，限 30 字符
slug = re.sub(r'[^\w\u4e00-\u9fff -]', '', slug_src)
slug = re.sub(r'\s+', '-', slug.strip())
slug = slug[:30].rstrip('-')
if not slug:
    slug = 'plan'

# 计算当日递增编号：扫描所有 YYYY-MM-DD-NNN-*.md 找最大编号
max_nnn = 0
prefix = f'{today}-'
for f in os.listdir(plans_dir):
    if f.startswith(prefix) and f.endswith('.md'):
        try:
            nnn_str = f[len(prefix):len(prefix)+3]
            max_nnn = max(max_nnn, int(nnn_str))
        except (ValueError, IndexError):
            pass

print(f'{today}-{max_nnn+1:03d}-{slug}.md')
" "$FILE_PATH" "$PROJECT_PLANS_DIR" 2>/dev/null)

if [ -z "$TARGET_NAME" ]; then
  # python3 失败时的 fallback：用原文件名
  TARGET_NAME="$(date +%Y-%m-%d)-001-${BASENAME}"
fi

cp "$FILE_PATH" "$PROJECT_PLANS_DIR/$TARGET_NAME"
echo "${BASENAME}=${TARGET_NAME}" >> "$MAPPING_FILE"

exit 0
