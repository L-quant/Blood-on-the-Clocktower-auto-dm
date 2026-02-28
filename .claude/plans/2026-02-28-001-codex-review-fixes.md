# Codex 代码 Review — 修复计划

## 背景

对 commits `9d96be7..533748e`（含 `3492b14` codex review fixes + `533748e` four core issues）进行代码审查，发现以下问题。这些代码由 Codex 生成，实现了 `2026-02-26-002-four-core-issues.md` 计划中的 4 个 Issue。

---

## 🔴 P0 — 运行时 Bug（影响功能正确性）

### BUG-A: `SquareView.vue:89` — `extend_time` 命令 payload key 错误

**文件**: `frontend/src/components/SquareView.vue:89`

```javascript
this.$store.commit("sendCommand", { type: "extend_time", payload: {} });
```

`websocket.js:730` 的 `sendCommand` 订阅读取的是 `payload.data`：
```javascript
ws.send(payload.type, payload.data || {});
```

应为 `data: {}` 而非 `payload: {}`。导致 `extend_time` 命令始终发送空数据。

---

### BUG-B: `PlayerActionSheet.vue:14` — 座位号显示为当前用户而非目标玩家（BUG-13 未修复）

**文件**: `frontend/src/components/PlayerActionSheet.vue:14`

```html
<span class="action-sheet__seat">{{ $t('square.seat', { n: seatIndex }) }}</span>
```

`seatIndex` 来自 `...mapState(["seatIndex"])`（当前用户座位号）。应使用 `targetSeatIndex`（目标玩家座位号）。同时 `seatIndexLabel` 计算属性已正确返回 `targetSeatIndex` 却未被使用。

---

### BUG-C: deadline 单位不一致 — `timer.set` vs `time.extended`

**后端**:
- `engine_extend.go:25`: `time.extended` deadline = `time.Now().Add(...).UnixMilli()` → **毫秒**
- `state.go:603`: `timer.set` 的 Reduce 做 `deadline * 1000` → 注释说 "convert unix seconds to millis"

**前端**:
- `websocket.js:562`: `timer.set` → `deadline * 1000`（假设后端发秒）
- `websocket.js:437`: `time.extended` → `deadline` 直接用（假设后端发毫秒）

两个事件对同一概念（deadline）使用不同单位，前后端虽然各自自洽但极易引发后续 bug。应统一为毫秒。

---

### BUG-D: `vote.js:45` — `currentYesCount` 仅在赞成票时重算

**文件**: `frontend/src/store/modules/vote.js:38-47`

```javascript
if (vote) {
  state.currentYesCount = state.votes.filter(v => v.vote).length;
}
```

当投票从 "yes" 改为 "no" 时，`currentYesCount` 不会减少。应无条件重算。

---

### BUG-E: `room_compose.go` vs `engine.go` 玩家计数逻辑不一致

**文件**:
- `room_compose.go:30-33`: 按 `SeatNumber > 0` 计数
- `engine.go:168-174`: 按 `!p.IsDM` 计数

如果 DM 有座位号（`SeatNumber > 0` 且 `IsDM == true`），composer 会多算一人；如果非 DM 玩家 `SeatNumber == 0`，composer 会少算一人。两种情况都可能导致 AI 生成错误数量的角色，引发 engine 的 `custom roles count N != player count M` 错误。

---

### BUG-F: `LobbyScreen.vue:182` — `starting` 标志设置后永不重置

**文件**: `frontend/src/components/LobbyScreen.vue:179-183`

```javascript
startGame() {
  if (!this.canStart || this.starting) return;
  this.starting = true;
  this.$store.dispatch("startGame");
},
```

如果 `startGame` 失败（命令被拒、WS 断开等），`starting` 永远为 `true`，"开始游戏"按钮永久禁用，只能刷新页面。

---

## 🟠 P1 — 规则违反（CLAUDE.md 代码红线）

### RULE-A: `engine.go` 禁止新增函数，但 `handleStartGame` 大幅膨胀

CLAUDE.md: "engine.go 1095 行为历史遗留，**禁止新增函数**，新功能拆到独立文件"

虽未新增顶层函数，但 `handleStartGame` 内新增了 ~45 行逻辑（custom_roles 解析 + no_action 自动完成 + actionType 填充），应提取到独立文件的 helper 函数中。

---

### RULE-B: 多个文件超 500 行限制

| 文件 | 行数 | 限制 |
|------|------|------|
| `engine.go` | 1063 | 500（历史遗留，禁止增长） |
| `state.go` | 743 | 500 |
| `autodm.go` | 966 | 500 |
| `websocket.js` | 735 | 300（Vuex 模块/插件） |

---

### RULE-C: 多个函数超 50 行限制（新增/膨胀部分）

因 Codex 变更而膨胀的函数：
- `handleStartGame` (engine.go): ~125 行（新增 custom_roles + actionType 逻辑）
- `handleAdvancePhase` (engine.go): ~85 行（新增 DM 权限检查）
- `_processGameEvent` (websocket.js): ~370 行

---

### RULE-D: `NewRoomActor` 10 个参数、`NewRoomManager` 7 个参数

CLAUDE.md: "函数参数 <= 4 个（超过用 struct 或 functional options）"

`NewRoomActor` 新增了 `composer game.Composer` 参数后达到 10 个。应使用配置结构体。

---

### RULE-E: 多处 `_ = json.Unmarshal(...)` 错误吞没

CLAUDE.md: "error 必须处理，禁止 `_ = err`"

新增的违规点：
- `engine.go:185`: `_ = json.Unmarshal(cmd.Payload, &payload)`
- `engine.go:188`: `_ = json.Unmarshal([]byte(cr), &customRoles)`
- `room_compose.go:64`: `_ = json.Unmarshal(cmd.Payload, &payload)`
- `room_compose.go:70`: `cmd.Payload, _ = json.Marshal(payload)`

---

### RULE-F: 嵌套超 3 层

- `engine.go` `handleAbility` starpass 分支：`for → switch → for → if → if` = 5 层
- `engine.go` `handleNomination` autodm 分支：`if → for → if` + break = 4 层

---

## 🟡 P2 — 代码质量问题

### QUAL-A: `engine.go:1060` `mustMarshal` 死代码

```go
func mustMarshal(v interface{}) []byte {
    b, _ := json.Marshal(v)
    return b
}
```

在 engine 包内无任何调用。应删除。

---

### QUAL-B: `websocket.js:151` 无意义三元表达式

```javascript
this.send('join', { name: this._store.state.playerId ? 'Player' : 'Player' });
```

两个分支返回相同字符串 `'Player'`，是死代码。

---

### QUAL-C: `autodm.go:405` 不安全类型断言

```go
event.PlayerID = nuid.(string)
```

如果 `nuid` 不是 string（如 JSON 数字），会 panic。应使用 comma-ok 断言。

---

### QUAL-D: `NightOverlay.vue` 和 `MeView.vue` catch 块未使用 error 变量

```javascript
} catch (e) {
  return '';
}
```

ESLint strict mode 下 `e` 未使用可能警告。

---

### QUAL-E: `VoteOverlay.vue:57` 硬编码中文

```html
{{ v.seatIndex }}号{{ v.vote ? '👍' : '👎' }}
```

`号` 应使用 i18n key。

---

### QUAL-F: `PlayerActionSheet.vue:122-124` — `seatIndexLabel` 计算属性从未使用

定义了但模板中从未引用。是 `seatIndex` → `targetSeatIndex` 修复的遗漏产物。

---

## 修复计划 Checklist

### P0 修复（必须立即修复）

- [x] **F1** 修复 `SquareView.vue:89` `payload: {}` → `data: {}`
- [x] **F2** 修复 `PlayerActionSheet.vue:14` `seatIndex` → `targetSeatIndex`，删除无用的 `seatIndexLabel` 计算属性和 `...mapState(["seatIndex"])`
- [x] **F3** 统一 deadline 单位为毫秒：
  - `state_reduce.go` timer.set 不再 `* 1000`（后端 bridge.go 已改用 `UnixMilli()`）
  - `ws_game_events.js` timer.set 不再 `* 1000`
- [x] **F4** 修复 `vote.js:45` `if (vote)` 守卫 → 无条件重算 `currentYesCount`
- [x] **F5** 修复 `room_compose.go:30-33` 玩家计数 → 改用 `!p.IsDM` 与 engine 保持一致
- [x] **F6** 修复 `LobbyScreen.vue` `starting` 标志：20s 超时自动重置 + beforeDestroy 清理

### P1 重构（应尽快修复）

- [x] **R1** 从 `handleStartGame` 提取 custom_roles 解析和 no_action 自动完成到 `engine_start_helpers.go`
- [x] **R2** `state.go` 的 `Reduce` 拆分到 `state_reduce.go`（~320 行，按事件类别分 handler）
- [x] **R3** `websocket.js` 的 `_processGameEvent` 拆分为 `ws_game_events.js`（~297 行）+ `ws_state_sync.js`（~85 行），websocket.js 缩减到 ~200 行
- [x] **R4** `NewRoomActor` / `NewRoomManager` 改用 `RoomDeps` 结构体（`room_config.go`）
- [x] **R5** 修复新增的 `_ = json.Unmarshal`：engine.go 改用 `parseCustomRoles()` 返回 error；room_compose.go 添加 error 日志

### P2 清理（后续清理）

- [x] **C1** 删除 `engine.go` 死代码 `mustMarshal`
- [x] **C2** 修复 `websocket.js` 无意义三元表达式 → 直接用 `'Player'`
- [x] **C3** 修复 `autodm.go` 类型断言 → comma-ok 模式
- [x] **C4** 修复 `NightOverlay.vue` / `MeView.vue` catch 块 → `catch (_e)`
- [x] **C5** 修复 `VoteOverlay.vue:57` 硬编码中文 → `$t('square.seat', { n: v.seatIndex })`
- [x] **C6** 删除 `PlayerActionSheet.vue` 无用 `seatIndexLabel`（已合并到 F2）

- [x] 回环检查：更新所有受影响的 CLAUDE.md 和文件头注释

## 状态：✅ 全部完成
