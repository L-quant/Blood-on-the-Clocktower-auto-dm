# 修复 Codex Review 5 项发现 + 规则润色 + 6 项防回归

## Context
上一版 game flow 修复（plan `2026-02-25-001`）已完成，后经 Codex review 发现 5 个遗留问题：
- P0：PhaseTimer 命令缺 IdempotencyKey，导致后续轮次超时被去重吞掉
- P1：超时链路在提名结算后断裂 + OwnerID 生命周期不完整
- P2：前端乐观写 myVote 导致投票被拒后 UI 锁死
- P3：AutoDM 仍从 ActorUserID 读 nominator，代理提名时读到 "autodm"

同时基于血染钟楼官方规则增加"每天仅允许一次处决"守卫。

**二次 Review 补充**：Codex 评审确认主线 5 个点已覆盖，额外建议 6 项防回归加固（标注 `[R#N]` 的为新增项）。

---

## 实现 Checklist

- [x] **步骤 1** Phase 1: PhaseTimer IdempotencyKey + 抗竞态保护
- [x] **步骤 2** Phase 1.5: 重启后计时器恢复
- [x] **步骤 3** Phase 2: nomination.resolved 超时 + room 层集成测试
- [x] **步骤 4** Phase 3: OwnerID 迁移 + DM 兜底
- [x] **步骤 5** Phase 4: AutoDM nominator 读取（含加严断言）
- [x] **步骤 6** Phase 5: 每天一次处决（含加严断言）
- [x] **步骤 7** Phase 6: 前端投票 UI 锁死（含请求关联 + 重置矩阵）
- [x] **步骤 8** Phase 7: 文档回环 + 验证
- [x] 回环检查：更新所有受影响的 CLAUDE.md 和文件头注释

---

## Phase 1: P0 — PhaseTimer IdempotencyKey 缺失 + 抗竞态保护

### 1.1 修复 PhaseTimer 命令构造
- `backend/internal/room/phase_timer.go` (line 49-57)
- 在 `CommandEnvelope` 构造中加入 `IdempotencyKey: uuid.NewString()`
- 已有 `uuid` import，无需新增依赖

```go
// 修复后:
cmd := types.CommandEnvelope{
    CommandID:      uuid.NewString(),
    IdempotencyKey: uuid.NewString(),
    RoomID:         pt.roomID,
    ...
}
```

### 1.2 [R#2] PhaseTimer 抗竞态保护（generation token）
- **问题**：`Schedule()` 中 `pt.timer.Stop()` 在回调已开始执行时返回 false，旧回调仍会 dispatch 陈旧超时命令
- **方案**：新增 `generation uint64` 字段，每次 Schedule 递增，回调中校验 generation 匹配才 dispatch

```go
type PhaseTimer struct {
    // ...existing fields...
    generation uint64
}

func (pt *PhaseTimer) Schedule(dur time.Duration, cmdType string, data map[string]string) {
    pt.mu.Lock()
    defer pt.mu.Unlock()

    if pt.timer != nil {
        pt.timer.Stop()
        pt.timer = nil
    }

    pt.generation++
    gen := pt.generation

    pt.timer = time.AfterFunc(dur, func() {
        pt.mu.Lock()
        if pt.generation != gen {
            pt.mu.Unlock()
            pt.logger.Debug("stale timer skipped", zap.Uint64("gen", gen))
            return
        }
        pt.mu.Unlock()
        // build and dispatch cmd (with IdempotencyKey)...
    })
}
```

### 1.3 添加单测
- 新建 `backend/internal/room/phase_timer_test.go`
- `TestPhaseTimerIdempotencyKey`：Schedule 后捕获 dispatched cmd，断言 `IdempotencyKey != ""`
- `TestPhaseTimerCancelPrevious`：连续 Schedule 两次，只有最后一次 fire
- `TestPhaseTimerGenerationGuard`：[R#2] 模拟旧回调在 Stop 后仍执行的场景，断言不 dispatch

---

## Phase 1.5: [R#1] 重启后计时器恢复

### 1.5.1 新增 recoverTimeoutFromState()
- **问题**：`NewRoomActor()` 只调 `loadState()` 恢复 state，不会按当前 phase/subphase 重新挂 timeout，服务重启后计时器丢失，游戏可能永久卡住
- **位置**：`backend/internal/room/room.go`，在 `loadState()` 之后、`go ra.loop()` 之前调用
- **实现**：

```go
// recoverTimeoutFromState re-schedules the appropriate phase timer
// after loading persisted state (e.g., after server restart).
func (ra *RoomActor) recoverTimeoutFromState() {
    if ra.state == nil || ra.state.Phase == "" || ra.state.Phase == "lobby" {
        return
    }
    cfg := ra.state.Config
    switch ra.state.Phase {
    case "first_night", "night":
        dur := time.Duration(cfg.NightActionTimeoutSec) * time.Second
        ra.phaseTimer.Schedule(dur, "advance_phase", map[string]string{"phase": "day"})
    case "day":
        switch ra.state.SubPhase {
        case "defense":
            dur := time.Duration(cfg.DefenseDurationSec) * time.Second
            ra.phaseTimer.Schedule(dur, "end_defense", nil)
        case "voting":
            dur := time.Duration(cfg.VotingDurationSec) * time.Second * time.Duration(len(ra.state.Players))
            ra.phaseTimer.Schedule(dur, "close_vote", nil)
        default: // discussion / post-nomination
            dur := time.Duration(cfg.DiscussionDurationSec) * time.Second
            ra.phaseTimer.Schedule(dur, "advance_phase", map[string]string{"phase": "night"})
        }
    }
    ra.logger.Info("recovered phase timer from state",
        zap.String("phase", ra.state.Phase),
        zap.String("sub_phase", ra.state.SubPhase))
}
```

- **注意**：恢复时使用完整超时时长（不做精确时间差计算），因为重启场景下宁可多等也不能误触发
- **调用点**：`NewRoomActor()` 中 `loadState()` 成功后立即调用

### 1.5.2 添加测试
- `backend/internal/room/room_timer_test.go`（或在 phase_timer_test.go 中追加）：
  - `TestRecoverTimeout_Night`：构造 phase=night 的 state → 恢复后断言 phaseTimer 有 pending advance_phase(day)
  - `TestRecoverTimeout_DayDiscussion`：phase=day, subPhase="" → 调度 advance_phase(night)
  - `TestRecoverTimeout_DayDefense`：phase=day, subPhase="defense" → 调度 end_defense
  - `TestRecoverTimeout_Lobby`：phase=lobby → 无计时器

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

### 2.2 [R#3] room 层 scheduleTimeouts 集成测试
- **问题**：计划原本只有 `phase_timer_test.go` 单元测试，但核心风险在 `scheduleTimeouts` 的分支逻辑（`room.go:296`）
- **位置**：新建 `backend/internal/room/schedule_timeouts_test.go`
- **测试用例**：
  - `TestScheduleTimeouts_NominationResolved_SchedulesAdvancePhase`：输入 `nomination.resolved` 事件 → 断言 phaseTimer 调度了 `advance_phase` 且 data.phase=night
  - `TestScheduleTimeouts_NominationResolved_ThenGameEnded_Cancels`：连续处理 `nomination.resolved` + `game.ended` → 断言 phaseTimer 已 Cancel（无 pending 计时）
  - `TestScheduleTimeouts_FullCycle`：`phase.day` → `nomination.created` → `defense.ended` → `nomination.resolved` → 验证每一步计时器正确切换

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

## Phase 4: P3 — AutoDM 代理提名 nominator 读取

### 4.1 convertEvent 优先读 payload 中的 nominator_user_id
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

### 4.2 [R#6a] 同步修复 event.PlayerID 赋值
- `autodm.go:396` 当前 `event.PlayerID = ev.ActorUserID` 在代理提名场景会设为 "autodm"
- 对于 `nomination.created`，应同样优先取 payload 中的 `nominator_user_id`：

```go
// line 396 area — 在 switch 结束后：
if ev.EventType == "nomination.created" {
    if nuid, ok := ev.Data["nominator_user_id"]; ok && nuid != "" {
        event.PlayerID = nuid
    }
}
event.Description = formatEventDescription(ev.EventType, event.Data)
```

### 4.3 添加测试（含加严断言 [R#6a]）
- `backend/internal/agent/autodm_test.go` 追加：
  - `TestConvertEvent_NominationWithNominatorUserID`：构造带 `nominator_user_id: "g2"` payload + `ActorUserID: "autodm"` → 断言：
    - `event.Data["nominator"] == "g2"`
    - **`event.PlayerID == "g2"`**（不是 "autodm"）
  - `TestConvertEvent_NominationDirectActor`：无 nominator_user_id 时 → 断言：
    - `event.Data["nominator"] == ev.ActorUserID`
    - `event.PlayerID == ev.ActorUserID`

---

## Phase 5: 规则润色 — 每天仅允许一次处决

### 5.1 resolveNomination 增加 ExecutedToday 守卫
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

### 5.2 添加测试（含加严断言 [R#6b]）
- `backend/internal/engine/vote_resolve_test.go` 追加：
  - `TestOnlyOneExecutionPerDay`：设 `state.ExecutedToday = "someone"`，投票过半 → 断言：
    - result 为 `"not_executed"`
    - 无 `player.died` 事件
    - **无 `execution.resolved` 事件**（`vote_resolve.go:73` 处只有 result=="executed" 才追加）
    - **无 `game.ended` 事件**（处决被守卫拦截时不应触发胜负判定）

---

## Phase 6: P2 — 前端投票被拒后 UI 锁死

### 6.1 移除乐观 myVote 写入
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

### 6.2 从 vote.cast 服务端事件确认 myVote
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
    // 服务端确认后才设置 myVote + 清 pending
    if (pe.actor_user_id === apiService.userId) {
        store.commit('vote/setMyVote', voteValue);
        store.commit('vote/setVotePending', false);
    }
    break;
}
```

### 6.3 添加 VoteOverlay 发送中状态防连点 + [R#5] 重置矩阵补齐
- `frontend/src/store/modules/vote.js` state 增加 `isVotePending: false`
- 添加 mutation `setVotePending(state, val)`
- `frontend/src/components/VoteOverlay.vue`:
  - `canVote` computed 增加 `&& !isVotePending` 条件
  - `castVote` 方法先 `commit('vote/setVotePending', true)` 再 dispatch sendVote
- **[R#5] 重置矩阵**——以下所有场景都必须将 `isVotePending` 清零：

| 触发点 | 位置 | 说明 |
|--------|------|------|
| `vote.cast`（自己的） | websocket.js | 正常确认路径 |
| `command_result` rejected（vote 命令） | websocket.js | 被拒路径 |
| `nomination.resolved` | websocket.js | 投票结算，整轮结束 |
| `endVote` mutation | vote.js | 手动结束投票 |
| **`startNomination` mutation** | vote.js:21 | **[R#5 新增]** 新提名开始，旧 pending 必须清 |
| **`resetVote` / module reset** | vote.js | **[R#5 新增]** 游戏重置/断线重连 |
| **WS disconnect** | websocket.js | **[R#5 新增]** 连接断开，所有 pending 无意义 |

### 6.4 [R#4] command_result 按 requestId→commandType 关联
- **问题**：原方案在任意 `command_result rejected` 时都清 `isVotePending`，会误伤其他命令（如 chat 被 rejected 也会清投票 pending）
- **方案**：维护 `pendingRequests: Map<requestId, commandType>`，只有 vote 类型被 reject 时才清 `isVotePending`

#### 6.4.1 websocket.js send() 改造
```javascript
send(command, data) {
    if (this._socket && this._socket.readyState === WebSocket.OPEN) {
        const requestId = Math.random().toString(36).substr(2);
        // 记录 pending 请求的命令类型
        this._pendingRequests = this._pendingRequests || {};
        this._pendingRequests[requestId] = command;
        this._socket.send(JSON.stringify({
            type: 'command',
            request_id: requestId,
            payload: {
                command_id: Math.random().toString(36).substr(2),
                room_id: this._roomId,
                type: command,
                data: data
            }
        }));
    }
}
```

#### 6.4.2 command_result handler
```javascript
case 'command_result': {
    let result = parsed.payload;
    if (typeof result === 'string') {
        try { result = JSON.parse(result); } catch(e) { break; }
    }
    const reqId = parsed.request_id || (result && result.request_id);
    const cmdType = this._pendingRequests && this._pendingRequests[reqId];
    if (reqId && this._pendingRequests) {
        delete this._pendingRequests[reqId];
    }
    if (result && result.status === 'rejected') {
        if (cmdType === 'vote') {
            store.commit('vote/setVotePending', false);
        }
        console.warn(`Command [${cmdType}] rejected:`, result.reason);
    }
    break;
}
```

#### 6.4.3 disconnect 时清空
- 在 `onClose` / `onError` handler 中：`this._pendingRequests = {}`
- 同时 `store.commit('vote/setVotePending', false)`

---

## Phase 7: 文档更新与回环检查

- [x] 更新 `backend/internal/room/CLAUDE.md`：补充 phase_timer_test.go、recoverTimeoutFromState、schedule_timeouts_test.go
- [x] 更新 `backend/internal/engine/CLAUDE.md`：补充 ExecutedToday 守卫、OwnerID 迁移说明
- [x] 更新 `backend/internal/agent/CLAUDE.md`：补充 nominator_user_id 优先逻辑 + PlayerID 修复
- [x] 更新 `frontend/src/store/CLAUDE.md`：补充 pendingRequests 机制
- [x] 更新 plan 文件状态

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
   - **路径 4**：[R#1] 后端重启（kill → 重启）→ 游戏中状态恢复 → 计时器继续工作

4. **超时场景**
   - 将 NominationTimeoutSec/DefenseDurationSec 降到 3-5 秒
   - 验证两轮完整循环中超时不被 dedup 吞掉

---

## 实现顺序

| 步骤 | Phase | 改动量 | 风险 | 新增/原有 |
|------|-------|--------|------|-----------|
| 1 | P0: IdempotencyKey + generation 保护 | +15 行 | 最高 — 不修会导致超时永久失效 | 原有 + [R#2] |
| 2 | 重启后计时器恢复 | +30 行 | 高 — 重启后游戏卡死 | [R#1] 新增 |
| 3 | P1a: nomination.resolved 超时 + room 层测试 | +5 行 + 测试 | 高 — 白天可能无限卡住 | 原有 + [R#3] |
| 4 | P1b: OwnerID 迁移 + DM 兜底 | +15 行 | 中 — 边缘场景房间不可用 | 原有 |
| 5 | P3: AutoDM nominator + PlayerID | +8 行 | 低 — 仅影响 AI 叙事准确性 | 原有 + [R#6a] |
| 6 | 规则: 每天一次处决 | +4 行 | 低 — 规则正确性提升 | 原有 + [R#6b] |
| 7 | P2: 前端投票 UI（含请求关联 + 重置矩阵） | ~40 行 | 中 — 需前端构建验证 | 原有 + [R#4][R#5] |
| 8 | 文档回环 | N/A | N/A | 原有 |

---

## 关键文件清单

| 文件 | 变更 |
|------|------|
| `backend/internal/room/phase_timer.go` | P0: IdempotencyKey + [R#2] generation 保护 |
| `backend/internal/room/phase_timer_test.go` | 新建: 计时器测试（含 generation guard） |
| `backend/internal/room/room.go` | P1a: scheduleTimeouts 加 nomination.resolved + [R#1] recoverTimeoutFromState |
| `backend/internal/room/schedule_timeouts_test.go` | [R#3] 新建: scheduleTimeouts 集成测试 |
| `backend/internal/engine/state.go` | P1b: player.left 迁移 OwnerID |
| `backend/internal/engine/engine.go` | P1b: advance_phase 增加 DM 兜底 |
| `backend/internal/engine/vote_resolve.go` | 规则: ExecutedToday 守卫 |
| `backend/internal/engine/vote_resolve_test.go` | 新增: OwnerID + 单日处决测试（[R#6b] 加严断言） |
| `backend/internal/agent/autodm.go` | P3: nominator_user_id + [R#6a] PlayerID 修复 |
| `backend/internal/agent/autodm_test.go` | 新增: convertEvent 测试（[R#6a] 加严 PlayerID 断言） |
| `frontend/src/store/index.js` | P2: 移除乐观 myVote |
| `frontend/src/store/modules/vote.js` | P2: 加 isVotePending + [R#5] 补齐重置点 |
| `frontend/src/store/plugins/websocket.js` | P2: vote.cast 确认 + [R#4] requestId→cmdType 映射 + [R#5] disconnect 清零 |
| `frontend/src/components/VoteOverlay.vue` | P2: canVote 加 pending 检查 |

---

## Review 建议追溯

| Review # | 建议 | 整合位置 |
|----------|------|---------|
| R#1 | 补"重启后计时器恢复" | Phase 1.5（新增整个 Phase） |
| R#2 | PhaseTimer 抗竞态保护 (generation) | Phase 1, section 1.2 |
| R#3 | 补 room 层 scheduleTimeouts 集成测试 | Phase 2, section 2.2 |
| R#4 | command_result 按 requestId 关联 | Phase 6, section 6.4 |
| R#5 | isVotePending 重置矩阵补齐 | Phase 6, section 6.3 |
| R#6 | 测试断言加严 (PlayerID + execution.resolved) | Phase 4 section 4.3 + Phase 5 section 5.2 |

## 状态：✅ 全部完成

> 注：本计划同步保存在项目目录 `.claude/plans/2026-02-26-001-codex-review-fixes.md`
