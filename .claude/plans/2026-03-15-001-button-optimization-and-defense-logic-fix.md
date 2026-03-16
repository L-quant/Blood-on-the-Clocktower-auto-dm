# 2026-03-15-001 流程按钮优化与辩护逻辑修正计划

## 背景
- 当前版本中，“延长讨论”按钮在某些场景下干扰了自动主持的流畅性。房主的“进入夜晚”按钮会莫名消失。
- 提名辩护逻辑存在串扰：提名者 A 和被提名者 B 的辩护阶段在前端显示上不够明确，导致非当前辩护玩家误操作（如 B 在 A 辩护时看到“结束辩护”按钮）。
- 目标：简化按钮交互，严格隔离提名/被提名双方的辩护流程，确保投票前的信息展示有序。

## 当日待完成项

### 1. 取消“延长讨论”按钮
- [ ] 移除前端所有涉及“延长讨论” (extend_discussion) 的按钮入口（通常在白天的广场或操作面板）。
- [ ] 若后端有对应命令处理逻辑，保持兼容但前端不再触发。

### 2. 恢复房主的“进入夜晚”按钮
- [ ] 检查并确保房主（Host）在白天结算完成后，能够正常看到“进入夜晚” (enter_night) 按钮。
- [ ] 修复当前版本该按钮意外消失的问题，确保测试阶段房主可以手动推进流程。

### 3. 辩护逻辑修正 (A 提名 B)
- [ ] **A 的辩护阶段（第一阶段）：**
    - [ ] 修正状态分发/前端显示：当进入 A 的辩护阶段时，所有玩家界面应明确显示当前是 ‘A（提名者）发表辩护’。
    - [ ] 按钮权限控制：此时被提名者 B **不应**显示“结束辩护”按钮。
    - [ ] 动作限制：只有 A 拥有“结束辩护”按钮。
- [ ] **B 的辩护阶段（第二阶段）：**
    - [ ] 状态衔接：当 A 点击“结束辩护”后，系统自动切换至 B 的辩护阶段。所有玩家界面应明确显示当前是 ‘B（被提名者）发表辩护’
    - [ ] 按钮切换：A 的“结束辩护”按钮消失，B 的窗口生成并显示“结束辩护”按钮。
- [ ] **流程推进：**
    - [ ] 只有在 B 点击“完成辩护”后，系统才正式开启所有玩家的投票序列。

### 4. 回环检查
- [ ] 检查所有受影响的 `CLAUDE.md` 和文件头注释。
- [ ] 验证 A/B 辩护顺序切换时，按钮的可见性是否完全符合预期，无权限越位。

## 状态：✅ 已完成

## 任务执行记录
1. **取消“延长讨论”按钮**：在 [SquareView.vue](frontend/src/components/SquareView.vue) 中移除了 `canExtendTime` 判断及其对应的 UI 按钮。
2. **恢复房主“进入夜晚”按钮**：修正了 [SquareView.vue](frontend/src/components/SquareView.vue) 中 `canAdvanceToNight` 的逻辑，使用 `$store.getters.isRoomOwner` 确保其可见性。
3. **辩护逻辑修正**：
   - 更新了 [vote.js](frontend/src/store/modules/vote.js) 状态管理，新增 `nominatorEnded` 和 `nomineeEnded`。
   - 增强了 [ws_game_events.js](frontend/src/store/plugins/ws_game_events.js)，支持 `defense.progress` 事件同步辩护进度。
   - 改造了 [VoteOverlay.vue](frontend/src/components/VoteOverlay.vue)，实现 A/B 辩护阶段的显示隔离与按钮权限控制。
   - 补充了中英双语 ([zh.json](frontend/src/i18n/zh.json), [en.json](frontend/src/i18n/en.json)) 的辩护文案。
