# Blood on the Clocktower Auto-DM — 四大核心问题需求与实现计划

## 背景（Context）

当前项目可以完成基本的游戏流程，但存在 4 个影响体验的核心问题：
1. 中文模式下大量内容仍显示英文
2. 夜晚行动缺乏角色差异化处理（信息类/选择类/无行动混为一谈）
3. 白天/夜晚自动推进不合理，缺少提名阶段和延时机制
4. 角色配置纯随机，缺乏真人说书人"配板子"的智能

本文档既是需求文档也是实现计划，每个 Issue 包含：现状分析 → 需求描述 → 实现 Checklist。

---

## Issue 1: 中文本地化不完整

### 现状分析

**已完成：**
- UI 标签/按钮通过 `$t()`/`$te()` 全覆盖，`zh.json` 有 20 个顶级 key
- `zh.json` 有 130+ 角色名翻译（`roles.washerwoman: "洗衣妇"` 等）
- 后端 `roles.go` 每个角色有 `NameCN` 和 `AbilityCN` 字段

**缺失：**
- **角色能力文本**：前端 `roles.json` 的 `ability` 字段只有英文，`zh.json` 无对应能力翻译。`NightOverlay.vue` 和 `MeView.vue` 展示能力时直接取 `roleData.ability`，永远英文
- **剧本描述**：`editions.json` 4 个剧本的 `description` 均为英文
- **传说角色**：`fabled.json` 中 `name` 和 `ability` 均为英文
- **角色提醒词**：`roles.json` 的 `reminders`、`firstNightReminder`、`otherNightReminder` 均英文
- **夜晚结果消息**：后端 `night.go` 的 `result.Message` 硬编码中文（当前对中文用户可用，但不支持切英文）
- **硬编码**：`HomeScreen.vue:11` 的 `alt="Blood on the Clocktower"`

### 需求描述

- 所有面向玩家的文本支持中英文切换
- 角色能力、剧本描述、传说角色能力需要中文翻译
- 前端夜晚结果展示根据当前语言显示本地化信息

### 实现计划

- [x] **1.1** 在 `zh.json` 中添加角色能力翻译：所有 Trouble Brewing 角色添加 `roles.{roleId}_ability` 键（文本来源：`roles.go` 的 `AbilityCN`）；同步在 `en.json` 中添加英文 `roles.{roleId}_ability` 键（来源：`roles.go` 的 `Ability`）
  - `frontend/src/i18n/zh.json`、`frontend/src/i18n/en.json`
  - 采用扁平 key `roles.washerwoman_ability` 而非嵌套结构，避免破坏现有 `$t('roles.washerwoman')` 用法

- [x] **1.2** 在 `zh.json`/`en.json` 中添加剧本描述翻译：`editions.{id}.name` 和 `editions.{id}.description`
  - `frontend/src/i18n/zh.json`、`frontend/src/i18n/en.json`

- [x] **1.3** 在 `zh.json`/`en.json` 中添加传说角色翻译：`fabled.{id}_name` 和 `fabled.{id}_ability`
  - `frontend/src/i18n/zh.json`、`frontend/src/i18n/en.json`

- [x] **1.4** 修改前端角色能力展示逻辑：ability 优先使用 `$t('roles.' + roleId + '_ability')`，`$te()` 检查后回退到 `roleData.ability`
  - `frontend/src/store/plugins/websocket.js`（`role.assigned` 和 `night.action.queued` 处理）
  - `frontend/src/components/NightOverlay.vue`（能力文本展示）
  - `frontend/src/components/MeView.vue`（角色信息面板）

- [x] **1.5** 修改夜晚结果展示：`night.action.completed` 事件传递结构化 `information` 数据到前端，前端根据 i18n 渲染结果文本，回退到后端 `result` 字符串
  - `frontend/src/store/plugins/websocket.js`（`night.action.completed` 处理）
  - `frontend/src/store/modules/night.js`（`setResult` 支持结构化数据）
  - `frontend/src/components/NightOverlay.vue`（result 步骤渲染）

- [x] **1.6** 修复 `HomeScreen.vue:11` 硬编码 alt 文本 → `:alt="$t('app.title')"`

- [x] **1.7** 回环检查：更新受影响的 CLAUDE.md 和文件头注释

---

## Issue 2: 缺少角色特定夜晚行动处理

### 现状分析

- **前端硬编码角色列表**：`websocket.js:444-458` 的 `selectOneRoles` 包含 46 个角色，信息类（washerwoman/librarian/chef/empath 等）和选择类混在一起。`selectTwoRoles` 只有 fortuneteller
- **`info` actionType 从未使用**：`night.js` 定义了 `'info'` 类型，`NightOverlay.vue` 和 `canSubmit` 也处理了它，但 websocket 永远不赋值 `'info'`
- **后端事件缺少 action_type**：`night.action.queued` payload 只有 `user_id`/`role_id`/`order`，前端不得不硬编码判断
- **Imp 首夜 UX 差**：前端给 Imp 显示选人界面，提交后被后端拒绝（`night.go:708`），玩家困惑
- **NightAction struct 无 ActionType 字段**：`state.go:73-80` 只有 UserID/RoleID/Order/Completed/TargetIDs/Result

**角色行动类型正确映射：**

| 角色 | 首夜 | 后续夜晚 |
|------|------|----------|
| washerwoman/librarian/investigator/chef | info | - |
| empath | info | info |
| fortuneteller | select_two | select_two |
| butler | select_one | select_one |
| undertaker | - | info |
| monk | - | select_one |
| ravenkeeper | - | select_one (死亡时) |
| spy | info | info |
| poisoner | select_one | select_one |
| imp | no_action | select_one |
| soldier/mayor/virgin/slayer/scarletwoman/baron/saint/recluse/drunk | (无 NightOrder) | (无 NightOrder) |

### 需求描述

- 后端 `night.action.queued` 事件包含 `action_type` 字段（info/select_one/select_two/no_action）
- 前端根据 `action_type` 决定 UI 模式，删除硬编码角色列表
- 信息类角色：显示"能力自动触发" → 自动提交 → 展示结果
- Imp 首夜：标记 `no_action`，不显示选人界面
- 选择类角色：正常选人界面

### 实现计划

- [x] **2.1** 在 `game/roles.go` 的 `Role` struct 添加 `FirstNightActionType string` 和 `NightActionType string` 字段，为所有 21 个 Trouble Brewing 角色填充正确值
  - `backend/internal/game/roles.go`
  - 可选值：`"info"` / `"select_one"` / `"select_two"` / `"no_action"` / `""`

- [x] **2.2** 在 `state.go` 的 `NightAction` struct 添加 `ActionType string` 字段
  - `backend/internal/engine/state.go:73-80`

- [x] **2.3** 修改 `engine.go` 生成 `night.action.queued` 事件时包含 `action_type`
  - `engine.go:244-249`（首夜循环）：查 role 的 `FirstNightActionType`
  - `engine.go:742-748`（后续夜晚循环）：查 role 的 `NightActionType`
  - Imp 首夜 `action_type="no_action"` → 立即生成 `night.action.completed`（result="首夜无行动"），不阻塞流程

- [x] **2.4** `state.go` Reduce 中 `night.action.queued` 事件解析 `action_type` 存入 NightAction
  - `backend/internal/engine/state.go`（Reduce 方法中 `night.action.queued` case）

- [x] **2.5** 前端 info 类角色自动提交：收到 `action_type="info"` 的 queued 事件后，NightOverlay 显示"你的能力自动触发"界面，玩家点确认 → 自动发送 `ability.use`（targets=[]）→ 等待结果
  - `frontend/src/store/plugins/websocket.js:432-483`（移除 selectOneRoles/selectTwoRoles 硬编码列表，改用 event payload 的 action_type）
  - `frontend/src/components/NightOverlay.vue`（info 类型流程优化）

- [x] **2.6** 前端 no_action 类型处理：显示"本夜无需行动"→ 确认后自动关闭
  - `frontend/src/components/NightOverlay.vue`

- [x] **2.7** 修改 `night_timeout.go` 的 `CompleteRemainingNightActions`：利用新的 `ActionType` 字段判断角色类型（为 Issue 3 的差异化超时做准备）
  - `backend/internal/engine/night_timeout.go`

- [x] **2.8** 添加后端测试：验证 night.action.queued 事件包含正确 action_type
  - `backend/internal/engine/engine_test.go`（3 个新测试）
  - `backend/internal/engine/night_timeout_test.go`（4 个新测试，含 isEvilCriticalAction 子测试）

- [x] **2.9** 回环检查：更新受影响的 CLAUDE.md 和文件头注释

---

## Issue 3: 白天/夜晚自动推进不合理

### 现状分析

- **白天直接跳夜晚**：`room.go:347` — `phase.day` 触发计时器 180s 后 `advance_phase("night")`，跳过提名
- **提名后快速入夜**：`room.go:358-359` — `nomination.resolved` 后 10s 即 `advance_phase("night")`
- **无延时机制**：玩家无法延长讨论时间
- **夜晚强制超时**：`room.go:342-343` — 30s 后 `advance_phase("day")`，`CompleteRemainingNightActions` 为所有未完成行动生成 `timed_out`。邪恶方关键选择（imp 杀人）不应被强制超时
- **提名系统存在但被绕过**：Nomination 相关结构和处理逻辑完整（SubPhaseDefense → SubPhaseVoting → resolved），但自动推进逻辑绕过了它
- **正确流程应为**：讨论（可延长，最多 3 次）→ 提名阶段 → 提名/投票循环 → 夜晚

### 需求描述

- 白天讨论计时器到期后 → 进入提名阶段（非直接入夜）
- 玩家可延长讨论时间，最多 3 次，每次 +60s
- 提名阶段有足够窗口允许多轮提名/投票
- 夜晚：信息类/善良选择类可超时自动完成；邪恶方关键行动（imp 杀人、poisoner 选毒）**无限等待**，仅周期性发提醒，永不强制超时
- 相关计时器恢复逻辑同步修改

### 实现计划

- [x] **3.1** `state.go` `GameConfig` 添加新字段；`State` 添加 `ExtensionsUsed int` 字段

- [x] **3.2** 白天讨论计时器到期目标改为提名阶段

- [x] **3.3** 提名阶段超时行为：`nomination.resolved` 后用 `NominationPhaseDurationSec` 而非 10s

- [x] **3.4** 新建 `engine_extend.go` 实现 `extend_time` 命令

- [x] **3.5** `state.go` Reduce 处理 `time.extended` 事件

- [x] **3.6** `room.go` `scheduleTimeouts` 处理 `time.extended` 事件

- [x] **3.7** 夜晚超时差异化处理（已在 Issue 2 步骤 2.7 完成）

- [x] **3.8** 修改夜晚计时器行为：night_timeout 命令 + engine_night_timeout.go + action.reminder 重调度

- [x] **3.9** 前端添加"延长时间"按钮 (SquareView.vue)

- [x] **3.10** 前端处理 `time.extended` 事件 (websocket.js + game.js)

- [x] **3.11** 前端提名阶段 UI：修复 canNominate 支持 nomination phase

- [x] **3.12** 同步修改 `recoverTimeoutFromState`：night→night_timeout, day→nomination, NominationPhaseDurationSec

- [x] **3.13** 添加后端测试：engine_extend_test.go (4 tests) + engine_night_timeout_test.go (3 tests)，修复 room 既有测试适配新行为

- [x] **3.14** 回环检查：更新 engine/CLAUDE.md, room/CLAUDE.md

---

## Issue 4: 角色组合缺乏智能（AI 配板子）

### 现状分析

- **纯随机选角**：`setup.go:234-258` 的 `selectRandomRoles` 从角色池随机取，无策略
- **Edition 参数被忽略**：`engine.go:180-183` 创建 `SetupConfig` 时未传 edition，所有游戏用同一 Trouble Brewing 池
- **Baron 是死代码**：`setup.go:93-102` 的 Baron 逻辑依赖 `BaronActive` 但 engine 从未设为 true
- **无 AI 参与**：AutoDM 完全不参与开局角色选择
- **SetupConfig 空字段**：`CustomRoles`、`BaronActive`、`Script` 均定义但未使用

### 需求描述

- AI（AutoDM）根据玩家人数 + 版本动态"配板子"，这是**必经步骤**，不是可选功能
- **房主也是玩家**，不能看到具体哪些角色会出现（不存在"房主审核"）
- 流程：房主点"开始游戏" → 后端自动调用 AI 组合角色 → 直接分配 → 游戏开始
- AI 考虑因素：游戏平衡、角色互动趣味性、玩家人数对应的分配规则
- Baron 被选中时正确修改外来者/镇民分配
- AI 调用失败时 fallback 到随机配置（保证游戏可开始）

### 实现计划

- [x] **4.1** 修复 Baron 死代码：角色选定后检查是否含 baron → 动态调整 outsider/townsfolk 数量
  - `backend/internal/game/setup.go`（`GenerateAssignments` 方法中选角后检查）
  - 无论 AI 选中还是随机选中 Baron，都正确触发 +2 outsider / -2 townsfolk

- [x] **4.2** 传递 Edition 到 SetupConfig：`engine.go:180-183` 从 `state.Edition` 读取
  - `backend/internal/engine/engine.go`

- [x] **4.3** 新建 `game/compose.go` 定义角色组合接口
  - `backend/internal/game/compose.go`（新文件）
  - `ComposeRequest`：PlayerCount, Edition
  - `ComposeResult`：Roles []string, Reasoning string（Reasoning 仅记录到 AI 日志，不展示给玩家）
  - `Composer` 接口：`ComposeRoles(ctx context.Context, req ComposeRequest) (*ComposeResult, error)`
  - `RandomComposer` 实现（重构现有 `selectRandomRoles` 逻辑）作为 fallback

- [x] **4.4** 实现 AI 角色组合器
  - `backend/internal/agent/subagent/composer.go`（新文件）
  - `AIComposer` 实现 `game.Composer` 接口
  - LLM Prompt 包含：
    - 玩家人数和标准分配规则（N 镇民 / M 外来者 / K 爪牙 / 1 恶魔）
    - 可用角色列表及能力描述（按版本过滤）
    - 要求考虑：信息类角色数量适中、有趣的角色互动组合（如 Poisoner+Empath、Drunk+Washerwoman）、不过于简单或复杂
    - 如果包含 Baron，自动标注需要修改外来者数量
  - 返回 JSON 格式角色 ID 列表
  - 超时（如 10s）或失败 → fallback 到 `RandomComposer`

- [x] **4.5** 集成 Composer 到游戏启动流程
  - 因为 engine 不能 import agent（架构边界），需要通过依赖注入：
    - `engine.go` 的 `handleStartGame` 接收一个 `game.Composer` 参数（或通过 SetupConfig 传入 `CustomRoles`）
    - 实际调用链：API/RoomActor 持有 Composer 引用 → `start_game` 命令到达时先调 Composer → 将结果作为 `CustomRoles` 传入 SetupConfig
  - `backend/internal/room/room.go`：RoomActor 持有 `game.Composer`，在 dispatch `start_game` 前异步调用
  - `backend/internal/game/setup.go`：`GenerateAssignments` 开头检查 `CustomRoles`，非空时直接使用（验证数量和类型正确性）

- [x] **4.6** 修改 `handleStartGame` 支持 AI 生成的角色列表
  - `backend/internal/engine/engine.go`
  - 从命令 payload 读取 `custom_roles`（JSON 数组，由 RoomActor 注入）
  - 传入 `SetupConfig.CustomRoles`
  - 游戏开始后将 AI 的 `Reasoning` 记录到 `AIDecisionLog`（仅后台可见，玩家不可见）

- [x] **4.7** 前端 `start_game` 流程调整：添加加载状态
  - `frontend/src/components/LobbyScreen.vue`
  - 房主点"开始游戏" → 显示"AI 正在配置角色..."加载状态
  - 收到 `game.started` 事件后进入游戏
  - 无需展示角色列表（房主也是玩家，不能提前看到）

- [x] **4.8** 添加后端测试
  - `backend/internal/game/setup_test.go`
  - 测试 Baron 修正、CustomRoles 使用、角色数量验证
  - 测试 AI Composer fallback 到 RandomComposer

- [x] **4.9** 回环检查：更新受影响的 CLAUDE.md 和文件头注释

---

## 实施顺序与依赖

```
Issue 2（夜晚行动） ──→ Issue 3（阶段推进）
       │                    │
       │  Issue 3.7 依赖 2.1 的 ActionType 字段
       │
Issue 1（本地化）    ──→ 可与 Issue 2 并行
Issue 4（智能配板）  ──→ 独立，最后实施
```

**分批交付顺序**：Issue 2 → Issue 3 → Issue 1 → Issue 4（每个 Issue 独立交付验证）

## 验证方式

1. **Issue 1**：切换语言为中文，检查所有页面无英文残留；切换英文确认回退正常
2. **Issue 2**：开始游戏后首夜，验证 Washerwoman 只看到结果不选人、Imp 无行动、Fortune Teller 选 2 人
3. **Issue 3**：白天讨论到期后进入提名阶段（非入夜）；延长 3 次后强制提名；夜晚 Imp 不选人时游戏无限阻塞而非自动天亮
4. **Issue 4**：房主点"开始游戏" → 短暂加载 → 游戏自动开始，角色由 AI 智能分配（非纯随机）；AI 失败时 fallback 随机仍可正常开始
5. **回归测试**：每个 Issue 交付后 `cd backend && make test` 全通过

---

## 浏览器全流程测试记录 (2026-02-27)

### 测试环境
- 后端: Go 服务 localhost:8080
- 前端: Vue dev server localhost:8081
- 玩家: 1 真人 (房主) + 6 bot (personality=random)
- 剧本: 暗流涌动 (Trouble Brewing), 7 人局

### 发现的问题

#### 🔴 P0 — 严重 (游戏破坏性)

**BUG-01: AI 聊天泄露所有玩家角色信息**
- 位置: AI 助手聊天频道 (公开可见)
- 现象: AI 自动生成的消息中直接写出所有玩家的角色、阵营信息
- 示例: "Player, your role is: Imp (Demon). Your Minion is: Frank"、"Charlie (Baron), you learn that Bob is the Demon"、"Alice (Soldier): Alive, Bob (Imp): Alive..."
- 原因: AutoDM AI 把本应是私密信息的内容发到了公开聊天频道，且 AI 不理解可见性边界
- 影响: 游戏完全无法正常进行，所有秘密信息公之于众

**BUG-02: 夜晚行动角色显示与实际角色不一致**
- 位置: NightOverlay 弹窗
- 现象: 我的角色是"士兵"(Soldier)，但夜晚弹出的行动窗口显示"共情者"(Empath) 并要求"选择你的目标"
- 原因: 士兵首夜没有行动 (无 NightOrder)，不应该收到夜晚行动弹窗。后端给我分配了错误的 night.action.queued 事件，或者前端 NightOverlay 使用了错误的角色数据
- 影响: 玩家困惑，被要求执行不属于自己角色的行动

**BUG-03: 士兵 (无夜间行动) 被要求选择目标**
- 位置: NightOverlay → 选择目标步骤
- 现象: Soldier 是被动角色，无首夜/后续夜晚行动，但系统仍弹出"选择你的目标"交互界面
- 对应 Issue 2: 缺乏角色特定夜晚行动类型处理

**BUG-04: 夜晚超时后结果显示 "timed_out" 原始字符串**
- 位置: NightOverlay 结果步骤
- 现象: 超时后弹窗显示原始文本 "timed_out" 而非友好的本地化消息
- 应显示: "你的行动已超时" 或 "夜晚行动自动完成"

#### 🟠 P1 — 重要 (体验缺陷)

**BUG-05: `teams.good` 未翻译 — 显示原始 i18n key**
- 位置: 左侧角色面板，角色名下方
- 现象: 显示 `teams.good` 而非 "善良阵营" 或 "好人"
- 原因: zh.json 中缺少 `teams.good` 翻译 key
- 对应 Issue 1

**BUG-06: 角色能力文本全英文**
- 位置: 左侧角色面板、NightOverlay 弹窗
- 现象: "You are safe from the Demon." (士兵能力)、"Each night, you learn how many of your 2 alive neighbours are evil." (共情者能力)
- 对应 Issue 1: 角色能力翻译缺失

**BUG-07: 恶魔伪装 (Bluffs) 显示英文 ID 而非中文名**
- 位置: 左侧面板 "恶魔伪装" 区域
- 现象: 显示 "slayer"、"virgin"、"saint" 而非 "杀手"、"贞女"、"圣徒"
- 原因: 前端直接展示 role_id 而非通过 i18n 或 rolesByKey 查找中文名
- 对应 Issue 1

**BUG-08: 白天阶段直接跳入夜晚，无提名阶段过渡**
- 位置: 白天 → 夜晚的自动推进
- 现象: 白天有"进入夜晚"按钮，直接跳入夜晚。投票结算后也自动进夜。没有独立的"提名阶段"
- 对应 Issue 3: 白天 → 提名 → 夜晚流程缺失

**BUG-09: AI 助手消息不区分公开/私密频道**
- 位置: 聊天面板
- 现象: 所有 AI 消息都显示在同一个"公开"频道中，包括本应是私密的夜晚信息
- 影响: 即使不考虑 AI 泄密问题，频道架构也有问题

**BUG-10: AI 消息过多且重复**
- 位置: AI 助手聊天频道
- 现象: 同一个事件 (如 game.started) 触发多条 AI 消息 (10+ 条)，内容高度重复
- 原因: 可能是 AutoDM 的多个子代理 (Moderator/Narrator/Rules 等) 各自独立响应同一事件
- 影响: 聊天被 AI 刷屏，真人消息被淹没

**BUG-11: Bot 玩家未投票**
- 位置: 提名投票阶段
- 现象: 提名后只有我 (1号) 投了赞成票，6个 bot 均未投票，最终只有 1 票
- 原因: Bot 的投票逻辑可能未触发，或 bot 的 WebSocket 连接不活跃
- 影响: 投票机制无法正常验证

#### 🟡 P2 — 一般 (UI/UX 细节)

**BUG-12: 标题栏始终显示英文 "Blood on the Clocktower"**
- 位置: 顶部导航栏
- 现象: 中文模式下标题仍为英文
- 对应 Issue 1.6

**BUG-13: 玩家操作面板座位号显示错误**
- 位置: 点击 2 号玩家后弹出的操作面板
- 现象: 面板标题显示 "1号" 而非 "2号"
- 原因: 可能是 seatIndex 偏移问题 (后端 1-indexed vs 前端 0-indexed)

**BUG-14: Bot 玩家在大厅无显示名字**
- 位置: 大厅座位列表
- 现象: Bot 座位只显示灰色圆点图标和"空"状态，无名字显示
- 实际: Bot 确实已入座（7/7），但视觉上不够明确

**BUG-15: 聊天 AI 助手频道残留 lobby 阶段消息**
- 位置: 进入游戏后的 AI 助手频道
- 现象: 游戏开始后仍可看到 "We currently have 2 players. We need at least 5 players..." 等 lobby 阶段的旧消息
- 建议: 游戏开始后清空或折叠 lobby 阶段的 AI 消息

**BUG-16: 投票结果"安全"无详细信息**
- 位置: 提名区域底部
- 现象: 仅显示 "安全" 二字，未显示 "1/4 票，未达到多数，被提名人安全"
- 建议: 补充投票详情

**BUG-17: 公开聊天频道被 AI 消息刷屏**
- 位置: 公开聊天频道 (非 AI 助手频道)
- 现象: AI 的叙事消息也出现在公开频道中，与 AI 助手频道内容重叠
- 建议: AI 叙事消息只出现在 AI 助手频道

### 与 4 大 Issue 的对应关系

| 发现 | Issue 1 (本地化) | Issue 2 (夜晚行动) | Issue 3 (阶段推进) | Issue 4 (配板) | 新问题 |
|------|:---:|:---:|:---:|:---:|:---:|
| BUG-01 AI泄密 | | | | | ✅ |
| BUG-02 角色错配 | | ✅ | | | |
| BUG-03 士兵被要求行动 | | ✅ | | | |
| BUG-04 timed_out原文 | ✅ | | | | |
| BUG-05 teams.good | ✅ | | | | |
| BUG-06 能力英文 | ✅ | | | | |
| BUG-07 伪装英文ID | ✅ | | | | |
| BUG-08 缺提名阶段 | | | ✅ | | |
| BUG-09 AI频道混乱 | | | | | ✅ |
| BUG-10 AI重复消息 | | | | | ✅ |
| BUG-11 Bot不投票 | | | | | ✅ |
| BUG-12 标题英文 | ✅ | | | | |
| BUG-13 座位号偏移 | | | | | ✅ |
| BUG-14 Bot无名字 | | | | | ✅ |
| BUG-15 lobby消息残留 | | | | | ✅ |
| BUG-16 投票结果简陋 | | | | | ✅ |
| BUG-17 AI刷屏公开频道 | | | | | ✅ |

### 结论

- **Issue 1 (本地化)** 的问题在测试中全面验证：能力文本、阵营标签、伪装 ID、结果字符串均有英文残留
- **Issue 2 (夜晚行动)** 问题严重：角色类型完全没有差异化，Soldier 被要求选目标
- **Issue 3 (阶段推进)** 确认：白天直接跳夜晚，无提名阶段过渡
- **Issue 4 (配板)** 未能直接验证（需要看 AI 是否参与选角），但角色组合看起来是随机的
- **新发现的 AI 泄密问题 (BUG-01) 是最严重的 P0 bug**，需要在 4 个 Issue 之前优先修复

### 测试截图 (2026-02-27)
- `/tmp/botc-test/01-home.png` — 首页
- `/tmp/botc-test/02-lobby.png` — 大厅 (1人)
- `/tmp/botc-test/03-lobby-full.png` — 大厅 (7人满)
- `/tmp/botc-test/04-game-start.png` — 游戏开始，NightOverlay 弹出
- `/tmp/botc-test/05-night-select.png` — 夜晚选择目标 + timed_out
- `/tmp/botc-test/06-day1.png` — 白天阶段
- `/tmp/botc-test/07-ai-chat-leak.png` — AI 聊天泄露角色信息
- `/tmp/botc-test/08-player-action.png` — 玩家操作面板
- `/tmp/botc-test/09-nomination.png` — 提名流程
- `/tmp/botc-test/10-vote.png` — 投票界面
- `/tmp/botc-test/11-vote-result.png` — 投票结果 + 夜晚

---

## 浏览器全流程测试记录 (2026-02-28)

### 测试环境
- 后端: Go 服务 localhost:8080 (make run-env)
- 前端: Vue dev server localhost:8081 (npm run serve)
- 玩家: 1 真人 (房主, 1号座) + 6 bot (API /bots 端点添加)
- 剧本: 暗流涌动 (Trouble Brewing), 7 人局
- 测试工具: Playwright MCP

### 发现的问题

#### 🔴 P0 — 严重 (游戏破坏性)

**BUG-01: AI 聊天泄露所有玩家角色信息 (依旧存在，且更严重)**
- 位置: 公开聊天频道 (非 AI 助手频道)
- 现象: AI 消息直接列出所有玩家角色和阵营：
  - 一条消息说 "Alice (Virgin), Frank (Imp), Bob (Investigator), Eve (Monk), Charlie (Scarlet Woman), Diana (Soldier), Player (Fortune Teller)"
  - 另一条说 "Charlie (Baron), Bob (Imp), Frank (Chef), Diana (Undertaker), Alice (Soldier), Player (Empath), Eve (Monk)"
  - 还有 "Player: **Imp** (Demon), Frank: **Poisoner** (Minion)"
- 新发现: **AI 各子代理给出完全不同的角色分配**，互相矛盾！说明 AI 在幻觉，不是读取真实游戏状态
- 影响: 游戏完全不可玩

**BUG-02: NightOverlay 显示错误角色 (依旧存在)**
- 位置: NightOverlay 弹窗
- 现象: 左面板显示我是"士兵"(Soldier)，但 NightOverlay 弹出"占卜师"(Fortune Teller) 图标和能力文本，要求"选择两个目标"
- 新发现: 士兵无首夜行动，不应收到任何夜晚弹窗
- 原因: 后端给 Soldier 发了 night.action.queued 事件，且 NightOverlay 使用了事件中的 role_id 而非玩家实际角色

**BUG-03: 无夜间行动角色被要求选择目标 (依旧存在)**
- 与 BUG-02 相同根因
- 对应 Issue 2: 缺乏角色特定夜晚行动类型处理

**BUG-04: 夜晚超时结果显示 "timed_out" 原始字符串 (依旧存在)**
- 位置: NightOverlay 结果步骤
- 现象: 弹窗标题"结果"，内容显示原始 "timed_out"
- 应显示: "夜晚行动已超时" 或本地化友好消息

**BUG-NEW-01: AI 各子代理生成互相矛盾的角色分配**
- 位置: 公开聊天频道，多条 AI 消息
- 现象: 同一局游戏中：
  - Moderator 说 Player 是 Imp, Frank 是 Poisoner
  - 另一个说 Player 是 Fortune Teller, Charlie 是 Scarlet Woman
  - 又一个说 Player 是 Empath, Charlie 是 Baron, Bob 是 Imp
- 原因: 各子代理 (Moderator/Narrator/Rules/Summarizer) 独立生成内容，不共享实际游戏状态，各自幻觉出不同的角色分配
- 影响: 即使不考虑泄密，AI 内容本身也完全不可信

**BUG-NEW-02: AI 天数计数严重错误**
- 位置: 公开聊天频道
- 现象: AI 消息显示 "Day 189", "Day 190", "Day 191"，而实际游戏只进行了 1-2 天
- 原因: AI 未正确读取 dayCount，可能是幻觉

#### 🟠 P1 — 重要 (体验缺陷)

**BUG-05: `teams.good` 未翻译 (依旧存在)**
- 位置: 左侧角色面板，角色名下方
- 现象: 显示原始 i18n key `teams.good`
- 控制台: `[vue-i18n] Value of key 'teams.good' is not a string`
- 对应 Issue 1

**BUG-06: 角色能力文本全英文 (依旧存在)**
- 位置: 左侧角色面板
- 现象: "You are safe from the Demon."
- 对应 Issue 1

**BUG-07: 恶魔伪装显示英文 ID (依旧存在)**
- 位置: 左侧面板"恶魔伪装"区域
- 现象: 显示 "librarian", "washerwoman", "saint" 而非中文名
- 对应 Issue 1

**BUG-08: 白天阶段直接跳入夜晚，无提名阶段过渡 (依旧存在)**
- 位置: 白天界面
- 现象: 白天有"进入夜晚"按钮，无独立"讨论→提名"阶段过渡
- 事件日志确认: 首夜→白天 (30s)→夜晚 (10s after nomination.resolved)→白天 (30s) — 循环极快
- 对应 Issue 3

**BUG-09: AI 消息全部发到公开频道，AI 助手频道为空 (加剧)**
- 位置: 聊天面板
- 新发现: "AI 助手"频道完全空白（显示"暂无消息"），所有 AI 消息（包括含敏感信息的）全部发到了"公开"频道
- 比上次更严重: 上次 AI 助手频道至少有内容，现在连 AI 助手频道都没用上

**BUG-10: AI 消息过多且重复 (依旧存在)**
- 位置: 公开聊天频道
- 现象: 公开频道有 20+ 条 AI 消息，每个阶段变更触发 5-6 条不同子代理的消息
- 刷屏严重，真人消息完全被淹没

**BUG-11: Bot 玩家未投票 (依旧存在)**
- 位置: 提名投票
- 现象: 提名后显示 "当前: 0 票，需要 4 票"，直接判定"安全"
- Bot 没有参与投票

**BUG-NEW-03: 士兵 (Soldier) 被 Imp 杀死**
- 位置: 事件日志
- 现象: 我的角色是士兵（"You are safe from the Demon"），但在第 1 夜后死亡
- 可能原因:
  1. Soldier 保护能力未正确实现
  2. 角色分配/投影可能有问题（AI 认为我是 Fortune Teller，也许引擎也这样认为）
  3. 死亡可能来自其他原因
- 需要进一步调查后端日志确认

**BUG-NEW-04: AI 引用错误的房间 ID**
- 位置: 公开聊天频道
- 现象: AI 消息中出现 "Room: 6864955f-0e3f-44f7-8e96-e2b4b5931158"，但当前房间是 "15f04e6a-997c-4109-9836-22c9bcf51b72"
- 原因: AI 可能在混淆不同房间的上下文

#### 🟡 P2 — 一般 (UI/UX 细节)

**BUG-12: 标题栏始终显示英文 (依旧存在)**
- 位置: 顶部导航栏
- 现象: "Blood on the Clocktower"（首页和大厅阶段）
- 游戏中标题正确显示阶段信息（"首夜"、"第1天·白天"等）
- 对应 Issue 1.6

**BUG-13: 玩家操作面板座位号偏移 (依旧存在)**
- 位置: 点击 2 号玩家弹出的操作面板
- 现象: 面板标题显示 "1号" 而非 "2号"
- 原因: seatIndex 偏移 (后端 1-indexed vs 前端 0-indexed)

**BUG-14: Bot 玩家在大厅无显示名字 (依旧存在)**
- 位置: 大厅座位列表
- 现象: Bot 座位只显示灰色圆点 "●"，无名字
- 实际: Bot 已入座 (7/7)

**BUG-15: lobby 阶段 AI 消息残留 (依旧存在)**
- 位置: 游戏开始后公开频道
- 现象: 仍可见 "We need at least 5 players to begin the game..." 等 lobby 消息
- 且全部英文

**BUG-16: 投票结果无详细信息 (依旧存在)**
- 位置: 提名区域底部
- 现象: 仅显示 "安全"，未显示票数详情

### 与 4 大 Issue + 新问题的对应关系

| 发现 | Issue 1 (本地化) | Issue 2 (夜晚行动) | Issue 3 (阶段推进) | Issue 4 (配板) | AI 系统问题 | UI/其他 |
|------|:---:|:---:|:---:|:---:|:---:|:---:|
| BUG-01 AI泄密 | | | | | ✅ | |
| BUG-02 角色错配 | | ✅ | | | | |
| BUG-03 无行动选目标 | | ✅ | | | | |
| BUG-04 timed_out原文 | ✅ | | | | | |
| BUG-05 teams.good | ✅ | | | | | |
| BUG-06 能力英文 | ✅ | | | | | |
| BUG-07 伪装英文ID | ✅ | | | | | |
| BUG-08 缺提名阶段 | | | ✅ | | | |
| BUG-09 AI频道空 | | | | | ✅ | |
| BUG-10 AI刷屏 | | | | | ✅ | |
| BUG-11 Bot不投票 | | | | | | ✅ |
| BUG-12 标题英文 | ✅ | | | | | |
| BUG-13 座位号偏移 | | | | | | ✅ |
| BUG-14 Bot无名字 | | | | | | ✅ |
| BUG-15 lobby残留 | | | | | ✅ | |
| BUG-16 投票简陋 | | | | | | ✅ |
| NEW-01 AI角色矛盾 | | | | | ✅ | |
| NEW-02 AI天数错 | | | | | ✅ | |
| NEW-03 士兵被杀 | | ✅? | | | | ✅ |
| NEW-04 AI房间ID错 | | | | | ✅ | |

### 结论

**与 2026-02-27 测试对比：所有 17 个已知 BUG 全部复现，另新增 4 个**：

1. **AI 系统问题是最严重的** — 不仅泄密，还各子代理互相矛盾（不同角色分配），天数计数错误（Day 190），引用错误房间 ID。根因是 AI 子代理未获得真实游戏状态，各自幻觉生成内容
2. **Issue 2 (夜晚行动)** 确认严重：Soldier 收到 Fortune Teller 的夜晚弹窗，且 Soldier 可能被 Imp 杀死
3. **Issue 3 (阶段推进)** 确认：首夜 30s 超时→白天→提名 10s 后→夜晚→30s→白天，循环极快无讨论时间
4. **Issue 1 (本地化)** 确认：teams.good 未翻译、能力英文、伪装英文 ID
5. **Issue 4 (配板)** 无法直接验证，但 AI 消息中各子代理给出的角色组合完全不同，说明随机选角后 AI 也不知道真实配置

### 优先级建议

修复顺序应为：
1. **AI 消息路由问题** — AI 消息应发到 AI 助手频道而非公开频道；敏感信息过滤
2. **Issue 2** — 角色夜晚行动类型差异化（阻塞游戏体验）
3. **Issue 3** — 阶段推进流程修复
4. **Issue 1** — 本地化完善
5. **Issue 4** — 智能配板

### 测试截图 (2026-02-28)
- `/tmp/botc-test-0228/01-home.png` — 首页
- `/tmp/botc-test-0228/02-lobby.png` — 大厅 (1人)
- `/tmp/botc-test-0228/03-lobby-full.png` — 大厅 (7人满)
- `/tmp/botc-test-0228/04-game-start-night.png` — 首夜，NightOverlay 弹出（显示 Fortune Teller 但我是 Soldier）
- `/tmp/botc-test-0228/05-night-select-targets.png` — 夜晚选择目标 + AI 泄密消息
- `/tmp/botc-test-0228/06-day1.png` — 白天阶段 + 公开频道 AI 泄密
- `/tmp/botc-test-0228/07-player-action-panel.png` — 玩家操作面板（座位号偏移 + AI 泄露完整角色列表）
- `/tmp/botc-test-0228/08-nomination.png` — 提名流程（0票安全 + AI 泄密加剧）
- `/tmp/botc-test-0228/09-ai-assistant-tab.png` — AI 助手频道（完全为空！）
- `/tmp/botc-test-0228/10-events-timeline.png` — 事件时间线（首夜→白天→死亡→投票→夜晚→白天）
- `/tmp/botc-test-0228/11-public-chat-full.png` — 公开聊天完整内容（AI 矛盾角色分配 + Day 190）
- `/tmp/botc-test-0228/12-whisper-tab.png` — 悄悄话频道（空）

## 状态：✅ 全部完成 - Issue 2 ✅ Issue 3 ✅ Issue 1 ✅ Issue 4 ✅
