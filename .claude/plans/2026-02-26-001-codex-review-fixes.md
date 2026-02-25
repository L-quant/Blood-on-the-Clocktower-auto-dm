# 修复 Codex Review 5 项发现 + 规则润色

## Context
上一版 game flow 修复（plan `2026-02-25-001`）已完成，后经 Codex review 发现 5 个遗留问题：
- P0：PhaseTimer 命令缺 IdempotencyKey，导致后续轮次超时被去重吞掉
- P1：超时链路在提名结算后断裂 + OwnerID 生命周期不完整
- P2：前端乐观写 myVote 导致投票被拒后 UI 锁死
- P3：AutoDM 仍从 ActorUserID 读 nominator，代理提名时读到 "autodm"

同时基于血染钟楼官方规则增加"每天仅允许一次处决"守卫。

---

## Phase 1: P0 — PhaseTimer IdempotencyKey 缺失

### 1.1 修复 PhaseTimer 命令构造
- `backend/internal/room/phase_timer.go` (line 49-57)
- 在 `CommandEnvelope` 构造中加入 `IdempotencyKey: uuid.NewString()`
- 已有 `uuid` import，无需新增依赖

```go
// 当前（缺失）:
cmd := types.CommandEnvelope{
    CommandID:   uuid.NewString(),
    RoomID:      pt.roomID,
    ...
}

// 修复后:
cmd := types.CommandEnvelope{
    CommandID:      uuid.NewString(),
    IdempotencyKey: uuid.NewString(),
    RoomID:         pt.roomID,
    ...
}
```

### 1.2 添加单测
- 新建 `backend/internal/room/phase_timer_test.go`
- `TestPhaseTimerIdempotencyKey`：Schedule 后捕获 dispatched cmd，断言 `IdempotencyKey != ""`
- `TestPhaseTimerCancelPrevious`：连续 Schedule 两次，只有最后一次 fire

---

## Phase 2: P1a — 提名结算后超时链路断裂

### 2.1 scheduleTimeouts 增加 nomination.resolved 分支
- `backend/internal/room/room.go` `scheduleTimeouts` (line 296-319)
- 在 `defense.ended` case 之后、`game.ended` case 之前，新增：

```go
case "nomination.resolved":
    // 投票结算后重新调度白天→夜晚超时，给出新提名窗口
    dur := time.Duration(cfg.NominationTimeoutSec) * time.Second
    ra.phaseTimer.Schedule(dur, "advance_phase", map[string]string{"phase": "night"})
```

- **原因**：`nomination.created` 会用 `end_defense` 覆盖白天计时器；结算后若无人再提名，白天永久卡住
- **不需要单独处理 `close_vote`**：`close_vote` 也会产出 `nomination.resolved` 事件，会命中此 case
- **与 `game.ended` 的优先级**：事件按顺序处理，若处决触发 `game.ended`，后面的 `Cancel()` 会覆盖此 Schedule，行为正确

---

## Phase 3: P1b — OwnerID 生命周期不完整

### 3.1 player.left 时迁移 OwnerID
- `backend/internal/engine/state.go` `Reduce` 的 `"player.left"` case (line 262-269)
- 在现有清理代码之后添加 owner 迁移逻辑：

```go
// 现有代码之后追加:
if s.OwnerID == event.Actor {
    s.OwnerID = ""
    for _, uid := range s.SeatOrder {
        if p, ok := s.Players[uid]; ok && !p.IsDM {
            s.OwnerID = uid
            break
        }
    }
}
```

- **SeatOrder 此时已移除离开者**，迭代安全
- player.left 仅在 PhaseLobby 允许（engine.go:113），不影响游戏中状态

### 3.2 advance_phase 增加 DM 兜底权限
- `backend/internal/engine/engine.go` `handleAdvancePhase` (line 685-689)
- 在 `isOwner` 之后增加 DM 检查，防止 OwnerID 为空的旧房间卡死：

```go
isAutoDM := cmd.ActorUserID == "autodm" || cmd.ActorUserID == "auto-dm"
isOwner := cmd.ActorUserID == state.OwnerID
isDM := false
if p, ok := state.Players[cmd.ActorUserID]; ok {
    isDM = p.IsDM
}
if !isAutoDM && !isOwner && !isDM {
    return nil, nil, fmt.Errorf("only room owner, DM, or autodm can advance phase")
}
```

### 3.3 添加测试
- `backend/internal/engine/vote_resolve_test.go` 追加：
  - `TestOwnerMigrationOnLeave`：3 人加入 → 第 1 人（owner）离开 → OwnerID 迁移到第 2 人
  - `TestOwnerMigrationAllLeave`：owner 离开后无非 DM 玩家 → OwnerID 为空
  - `TestDMCanAdvancePhase`：DM 可调用 advance_phase（即使 OwnerID 为空）

---

## Phase 4: P2 — 前端投票被拒后 UI 锁死

### 4.1 移除乐观 myVote 写入
- `frontend/src/store/index.js` `sendVote` action (line 295-301)
- 删除 `commit('vote/setMyVote', vote)` 这行
- 改为仅发送命令，等 server 确认后再更新：

```javascript
sendVote({ commit }, vote) {
    // 不再乐观写入 — 等待 vote.cast 服务端确认
    commit('sendCommand', {
        type: 'vote',
        data: { vote: vote ? 'yes' : 'no' }
    });
},
```

### 4.2 从 vote.cast 服务端事件确认 myVote
- `frontend/src/store/plugins/websocket.js` `vote.cast` handler (line 369-378)
- 追加自身投票检测：

```javascript
case 'vote.cast': {
    const voterSeat = parseInt(eventData.voter_seat, 10) || 0;
    const voteValue = eventData.vote === 'yes';
    store.commit('vote/castVote', {
        seatIndex: voterSeat,
        vote: voteValue
    });
    store.commit('vote/setCurrentVoter', voterSeat);
    // 服务端确认后才设置 myVote
    if (pe.actor_user_id === apiService.userId) {
        store.commit('vote/setMyVote', voteValue);
    }
    break;
}
```

### 4.3 添加 VoteOverlay 发送中状态防连点
- `frontend/src/store/modules/vote.js` state 增加 `isVotePending: false`
- 添加 mutation `setVotePending(state, val)`
- `frontend/src/components/VoteOverlay.vue`:
  - `canVote` computed 增加 `&& !isVotePending` 条件
  - `castVote` 方法先 `commit('vote/setVotePending', true)` 再 dispatch sendVote
- `vote.cast` handler 中（websocket.js）：当 `pe.actor_user_id === apiService.userId` 时同时 `commit('vote/setVotePending', false)`
- `nomination.resolved` handler 和 `endVote` mutation 中重置 `isVotePending = false`

### 4.4 command_result 处理被拒命令
- `frontend/src/store/plugins/websocket.js` `command_result` case (line 158-160)
- 解析 payload 并在被拒时重置 pending 状态：

```javascript
case 'command_result': {
    let result = parsed.payload;
    if (typeof result === 'string') {
        try { result = JSON.parse(result); } catch(e) { break; }
    }
    if (result && result.status === 'rejected') {
        // 重置投票 pending 状态（最常见的被拒场景）
        store.commit('vote/setVotePending', false);
        console.warn('Command rejected:', result.reason);
    }
    break;
}
```

---

## Phase 5: P3 — AutoDM 代理提名 nominator 读取

### 5.1 convertEvent 优先读 payload 中的 nominator_user_id
- `backend/internal/agent/autodm.go` `convertEvent` (line 381-383)
- 修改：

```go
case "nomination.created":
    event.Type = "nomination"
    // Prefer explicit nominator from payload (autodm proxy case)
    if nuid, ok := event.Data["nominator_user_id"]; ok && nuid != "" {
        event.Data["nominator"] = nuid
    } else {
        event.Data["nominator"] = ev.ActorUserID
    }
```

- `event.Data` 类型为 `map[string]interface{}`，`nominator_user_id` 来自 payload 解析（line 362-363），值为 `string`
- 同样更新 `event.PlayerID` 设置（line 396），使其也取正确的 nominator

### 5.2 添加测试
- `backend/internal/agent/autodm_test.go` 追加：
  - `TestConvertEvent_NominationWithNominatorUserID`：构造带 `nominator_user_id: "g2"` payload 的 nomination.created 事件 + `ActorUserID: "autodm"`，断言转换后 `event.Data["nominator"] == "g2"`
  - `TestConvertEvent_NominationDirectActor`：无 nominator_user_id 时回退到 ActorUserID

---

## Phase 6: 规则润色 — 每天仅允许一次处决

### 6.1 resolveNomination 增加 ExecutedToday 守卫
- `backend/internal/engine/vote_resolve.go` `resolveNomination` (line 59-62)
- 在判定 `result = "executed"` 之后追加检查：

```go
result := "not_executed"
if yesVotes >= threshold {
    result = "executed"
}
// 官方规则：每天仅允许一次处决
if result == "executed" && state.ExecutedToday != "" {
    result = "not_executed"
}
```

- `state.ExecutedToday` 在 `phase.day` Reduce 时清零（state.go:355），确保每天重置
- 提名仍可继续（BotC 规则允许多次提名），但处决不会再次发生

### 6.2 添加测试
- `backend/internal/engine/vote_resolve_test.go` 追加：
  - `TestOnlyOneExecutionPerDay`：设 `state.ExecutedToday = "someone"`，投票过半 → 断言 result 为 "not_executed"，无 `player.died` 事件

---

## Phase 7: 文档更新与回环检查

- [ ] 更新 `backend/internal/room/CLAUDE.md`：补充 phase_timer_test.go
- [ ] 更新 `backend/internal/engine/CLAUDE.md`：补充 ExecutedToday 守卫说明
- [ ] 更新 `backend/internal/agent/CLAUDE.md`：补充 nominator_user_id 优先逻辑
- [ ] 更新 plan 文件状态

---

## 验证方案

1. **后端单测**
   ```bash
   cd backend && go test ./internal/engine/... ./internal/room/... ./internal/agent/... -v -count=1
   ```

2. **前端构建**
   ```bash
   cd frontend && npm run lint-ci && npm run build
   ```

3. **集成冒烟测试**
   - 启动前后端，5 标签页模拟 5 玩家
   - 路径 1：首夜 → 白天 → 提名 → 投票（被拒场景：Butler 限制）→ 验证按钮恢复
   - 路径 2：提名 → 结算 → 等待 NominationTimeoutSec → 验证自动进入夜晚
   - 路径 3：处决恶魔 → game.ended → 重开新局 → 验证超时仍正常

4. **超时场景**
   - 将 NominationTimeoutSec/DefenseDurationSec 降到 3-5 秒
   - 验证两轮完整循环中超时不被 dedup 吞掉

---

## 实现顺序

| 步骤 | Phase | 改动量 | 风险 |
|------|-------|--------|------|
| 1 | P0: IdempotencyKey | +2 行 | 最高 — 不修会导致超时永久失效 |
| 2 | P1a: nomination.resolved 超时 | +3 行 | 高 — 白天可能无限卡住 |
| 3 | P1b: OwnerID 迁移 + DM 兜底 | +15 行 | 中 — 边缘场景房间不可用 |
| 4 | P3: AutoDM nominator 读取 | +5 行 | 低 — 仅影响 AI 叙事准确性 |
| 5 | 规则: 每天一次处决 | +4 行 | 低 — 规则正确性提升 |
| 6 | P2: 前端投票 UI 锁死 | ~20 行 | 中 — 需前端构建验证 |
| 7 | 文档回环 | N/A | N/A |

---

## 关键文件清单

| 文件 | 变更 |
|------|------|
| `backend/internal/room/phase_timer.go` | P0: 加 IdempotencyKey |
| `backend/internal/room/phase_timer_test.go` | 新建: 计时器测试 |
| `backend/internal/room/room.go` | P1a: scheduleTimeouts 加 nomination.resolved |
| `backend/internal/engine/state.go` | P1b: player.left 迁移 OwnerID |
| `backend/internal/engine/engine.go` | P1b: advance_phase 增加 DM 兜底 |
| `backend/internal/engine/vote_resolve.go` | 规则: ExecutedToday 守卫 |
| `backend/internal/engine/vote_resolve_test.go` | 新增: OwnerID + 单日处决测试 |
| `backend/internal/agent/autodm.go` | P3: 优先读 nominator_user_id |
| `backend/internal/agent/autodm_test.go` | 新增: convertEvent 测试 |
| `frontend/src/store/index.js` | P2: 移除乐观 myVote |
| `frontend/src/store/modules/vote.js` | P2: 加 isVotePending |
| `frontend/src/store/plugins/websocket.js` | P2: vote.cast 确认 + command_result 处理 |
| `frontend/src/components/VoteOverlay.vue` | P2: canVote 加 pending 检查 |

## 状态：📋 待审核

> 注：本计划同步保存在项目目录 `.claude/plans/2026-02-26-001-codex-review-fixes.md`
