# INTERVIEW PREP — Blood on the Clocktower Auto-DM

> 更新时间：2026-02-11
> 审查范围：`backend/cmd/server`, `backend/internal/*`（agent, engine, room, store, realtime, projection, mcp, rag, queue, game, types）
> 视角：高级后端架构师面试官 + LLM Ops 专家

---

## 1. Architecture & Code Audit（架构与代码审计）

### 1.1 整体架构一览

```
┌─────────────────────────────────────────────────────────────────────┐
│                         HTTP / WebSocket 层                         │
│   api.Server (Chi)          realtime.WSServer (Gorilla WS)         │
│   REST: /v1/auth, /rooms    WS: subscribe, command, ping           │
├─────────────────────────────────────────────────────────────────────┤
│                         Room Actor 层                               │
│   RoomManager → RoomActor (cmdCh chan, single goroutine loop)      │
│   命令串行 → engine.HandleCommand → AppendEvents → broadcast       │
├─────────────────────────────────────────────────────────────────────┤
│                         确定性引擎层                                │
│   engine.HandleCommand (纯函数，无副作用)                           │
│   engine.State.Reduce (事件驱动状态演进)                            │
│   game.SetupAgent / game.NightAgent (角色分配 / 夜间结算)          │
├─────────────────────────────────────────────────────────────────────┤
│                         AI Agent 层                                 │
│   AutoDM → core.Orchestrator → 5 Sub-Agents                       │
│   MCP Registry (6 tools, JSON Schema validation)                   │
│   RAG (Qdrant + OpenAI Embedding → injectRuleContext)              │
│   RabbitMQ Task Queue (priority + DLQ + retry)                     │
├─────────────────────────────────────────────────────────────────────┤
│                         持久化层                                    │
│   store.Store (MySQL)                                               │
│   events (append-only) + snapshots + commands_dedup + room_sequences│
│   Qdrant (向量数据库，规则文档索引)                                  │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 简历声明 vs 代码实现逐项审计

#### A. Multi-Agent Orchestration with FSM

**简历声明**："Multi-Agent 系统，Orchestrator + Specialists 架构，FSM 状态编排"

**代码实现**：

1. **Orchestrator + Specialists 存在且可用** ✅
   - `internal/agent/core/orchestrator.go` 是 AutoDM 实际使用的 Orchestrator
   - 5 个 Sub-Agent 全部实现：`Moderator`、`Narrator`、`Rules`、`Summarizer`、`PlayerModeler`
   - 事件路由在 `routeEvent()` (:182-197)，按 event.Type 分发到对应 handler
   - 每个 handler 调用对应的 sub-agent（如 `handlePhaseChange` → `narrator.NarratePhaseChange`）

2. **双 Orchestrator 实现存在冗余** ⚠️
   - `internal/agent/orchestrator.go` 是另一套独立实现（polling-based，使用 `zap.Logger`）
   - `internal/agent/core/orchestrator.go` 是 AutoDM 实际使用的（event-driven，使用 `slog.Logger`）
   - 前者定义了 7 步控制循环（Sense → BuildContext → Plan → ExecuteActions → Observe → Reflect → PersistMemory），但并未被 AutoDM 调用
   - **面试风险**：被问到"你的 7 步控制循环怎么工作的"时，需要解释这是设计文档级实现，实际运行路径是 event-driven 的 core.Orchestrator

3. **FSM 实现方式** ⚠️
   - 规则层 FSM：`engine/state.go` 的 `Phase`/`SubPhase` 枚举转换（:8-28），由 `Reduce()` 方法驱动
   - 状态转换散落在 `Reduce()` 的 switch-case 中（:203-469），不是显式 FSM 表
   - **没有使用 FSM 库或显式状态转换表**，是隐式 FSM
   - Agent 层无独立 FSM，靠 `routeEvent` 的 switch 做事件分发

**对应代码路径**：
```
AutoDM.OnEvent (autodm.go:327)
  → updateGameStateFromEngineState (autodm.go:947)
  → publishAsyncTask / ProcessQueuedEvent (autodm.go:372/347)
    → convertEvent → injectRuleContext → ProcessEvent (autodm.go:214)
      → core.Orchestrator.ProcessEvent (core/orchestrator.go:165)
        → routeEvent → handlePhaseChange/handleNomination/... (core/orchestrator.go:182)
          → narrator.NarratePhaseChange / moderator.Process / ... (subagent/*.go)
```

#### B. Event Sourcing with Snapshots and last_seq

**简历声明**："Event Sourcing 架构，append-only 事件流，快照 + 增量同步，last_seq 断线重连"

**代码实现**：✅ **高度吻合，是项目最扎实的部分**

1. **Append-only 事件流**：
   - `store/event_store.go:109-151`：`AppendEvents()` 使用 MySQL 事务 + `room_sequences FOR UPDATE` 保证原子递增 seq
   - 事件表 PK 为 `(room_id, seq)`，天然不可变

2. **快照**：
   - `room/room.go:222-230`：当 `nextState.LastSeq % snapshotInterval == 0` 时生成快照
   - 快照使用应用后状态 `nextState`（而非当前状态），避免与事件尾部不一致

3. **状态重建**：
   - 热路径：`loadState()` (room.go:80-107) = `GetLatestSnapshot` + `LoadEventsAfter`
   - 回放路径：`replay()` (api.go:326-353) = `LoadEventsUpTo` + 全量 Reduce

4. **last_seq 增量同步**：
   - WS `handleSubscribe()` (ws.go:196-245)：客户端携带 `last_seq`，服务端 `LoadEventsAfter` 补发
   - HTTP `fetchEvents()` (api.go:266-280)：支持 `after_seq` 查询参数

5. **幂等去重**：
   - `commands_dedup` 表 + `GetDedupRecord()` (event_store.go:21-31)
   - 命中时直接返回缓存的 `CommandResult`，不写新事件

#### C. Actor Model with Channel Serialization

**简历声明**："Actor Model，每房间独立 Goroutine，Channel 串行化处理命令"

**代码实现**：✅ **核心机制正确，但不是纯 Channel 模型**

1. **Channel 串行化** ✅：
   - `RoomActor.cmdCh chan CommandRequest` 缓冲 256 (room.go:47,67)
   - `loop()` (room.go:120-145) 单 goroutine 消费 cmdCh
   - 所有命令通过 `Dispatch()` (room.go:288-302) → cmdCh → loop → handleCommand 串行执行

2. **非纯 Channel 模型** ⚠️：
   - `stateMu sync.RWMutex`：保护 `GetState()` 的并发读 (room.go:311-315)
   - `subsMu sync.RWMutex`：保护订阅者 map 的并发读写 (room.go:276-286)
   - 这是合理的工程选择（读多写少场景），但简历不应说"完全无锁"

3. **Crash Recovery** ✅：
   - `executeCommand()` (room.go:147-161) 捕获命令级 panic，标记 fatal
   - `loop()` (room.go:120-145) 捕获致命 panic，调用 `onCrash` 回调
   - `handleActorCrash()` (room.go:364-378) 重建 actor，从 DB 重新 hydrate

4. **Context 生命周期正确** ✅：
   - Room actor 的 ctx 来自 `RoomManager` 级别 (room.go:333)，不与 HTTP request 绑定

#### D. RAG with Dynamic Context Injection

**简历声明**："RAG 动态上下文注入，Qdrant 向量数据库，降低 LLM 幻觉概率"

**代码实现**：✅ **链路完整，但 RAG 质量取决于文档质量**

1. **Qdrant + OpenAI Embedding**：
   - `rag/client.go`：QdrantClient HTTP API（EnsureCollection, Upsert, Search）
   - `rag/embedding.go`：OpenAIEmbedding 使用 `/v1/embeddings` 接口
   - `rag/retriever.go`：RuleRetriever 负责加载 markdown 规则、分块、嵌入、检索

2. **动态注入**：
   - `autodm.go:441-482`：`injectRuleContext()` 在事件处理前调用 retriever
   - `buildRuleQuery()` (autodm.go:484-503)：按事件类型生成模板化查询
   - 检索结果注入 `event.Data["rule_context"]` 和 `event.Description`
   - 超时控制 1500ms (autodm.go:457)

3. **降级设计** ✅：
   - retriever 为 nil 时跳过 (autodm.go:448-449)
   - 检索失败不影响主流程 (autodm.go:461)

#### E. MCP with JSON Schema Validation

**简历声明**："MCP 协议标准化工具访问，JSON Schema 参数校验"

**代码实现**：✅ **实现完整，是亮点之一**

1. **Registry + Handler 模式**：
   - `mcp/registry.go`：Registry 管理 ToolDefinition + ToolHandler
   - `Invoke()` (registry.go:158-206)：查找 → 校验参数 → 执行 handler
   - 支持同步和异步两种模式 (invokeAsync, registry.go:208-253)

2. **JSON Schema 校验** ✅：
   - `validateParams()` (registry.go:268-289)：检查 required 字段
   - `validateValue()` (registry.go:291-364)：递归校验 string/number/boolean/array/object
   - 支持 Enum、MinLength、MaxLength、Minimum、Maximum、Pattern、Items、Properties

3. **AutoDM 注册了 6 个工具** (autodm.go:605-928)：
   - `send_public_message`：公开发言
   - `send_private_message`：私聊
   - `request_player_confirmation`：请求确认（同时写审计事件）
   - `toggle_voting`：切换投票模式
   - `advance_phase`：推进阶段
   - `write_event`：写入审计事件

4. **mcp/tools.go 注册了 11 个工具**（供更大编排链路使用）

5. **Auditor** (registry.go:378-416)：记录所有工具调用的审计日志

#### F. EINO 框架

**简历声明（如有）**："基于 EINO 框架"

**代码实现**：❌ **代码中无任何 EINO 引用**

- 整个 codebase 中无 `eino` import
- Agent 系统完全自建（`agent/core/`, `agent/subagent/`, `agent/llm/`, `agent/memory/`, `agent/tools/`）
- LLM 调用直接使用 OpenAI-compatible HTTP API (`agent/llm.go`, `agent/llm/router.go`)

#### G. DDD（领域驱动设计）

**简历声明**："DDD 驱动的领域建模"

**代码实现**：⚠️ **有 DDD 影子，但不是严格 DDD**

1. **聚合边界隐式存在**：
   - `RoomActor` + `engine.State` 可视为房间聚合根
   - 命令统一通过 `RoomActor.handleCommand` 入口
   - 不变量集中在 `engine.HandleCommand` 中

2. **缺少显式 DDD 模式**：
   - 没有显式 Aggregate Root 接口
   - 没有 Value Object 标注
   - 没有 Repository 抽象（直接使用 `store.Store`）
   - 没有 Domain Event 与 Integration Event 区分

3. **建议说法**："领域模型与聚合边界已明确，采用 DDD 思想指导了模块划分"

### 1.3 代码质量亮点

| 亮点 | 代码位置 | 说明 |
|------|----------|------|
| 事务原子性 | `store/event_store.go:109-151` | events + dedup + snapshot 在同一事务 |
| Deep Copy | `engine/state.go:141-201` | State.Copy() 完整深拷贝，避免共享状态 |
| Panic Recovery 分级 | `room/room.go:120-161` | 命令级 vs actor级 panic 分开处理 |
| Visibility Projection 三层 | `projection/projection.go` | allowed → sanitizePayload → ProjectedState |
| LLM Timeout + Fallback | `autodm.go:347-370` | context.WithTimeout + defaultMessageForEvent |
| 自激回路防护 | `autodm.go:331-334` | 过滤 autodm 自身产生的事件 |
| WS Rate Limiting | `realtime/ws.go:305-332` | Token Bucket 限流 |
| Night Ability 全覆盖 | `game/night.go` | TB 全部角色能力实现，含中毒/醉酒失灵逻辑 |

### 1.4 潜在改进点

| 问题 | 位置 | 影响 | 建议 |
|------|------|------|------|
| 双 Orchestrator 并存 | `agent/orchestrator.go` vs `agent/core/orchestrator.go` | 代码冗余，面试追问风险 | 删除未使用的 `agent/orchestrator.go` 或明确其用途 |
| FSM 隐式散落 | `engine/state.go:Reduce()` | 状态转换不可视化 | 可引入状态转换表或使用 FSM 库 |
| Memory 实现存根 | `agent/memory.go` SimpleEmbedder | Hash 伪嵌入，非真实向量 | 明确为测试用途，文档标注 |
| LastSeenSeq 未使用 | `types.go` CommandEnvelope.LastSeenSeq | 乐观并发控制未落地 | 在 handleCommand 中对比并返回冲突 |
| Error 吞没 | 多处 `_ = json.Unmarshal(...)` | 静默忽略解析错误 | 关键路径应至少 log.Warn |
| CORS 全开 | `api.go:104` `Access-Control-Allow-Origin: *` | 生产安全隐患 | 改为配置化 allowed origins |

### 1.5 审计发现 & 已修复 Bug

| Bug | 文件 & 行 | 影响 | 修复 |
|-----|-----------|------|------|
| Role ID 不匹配：`"fortuneteller"` vs canonical `"fortune_teller"` | `game/night.go:111` (ResolveAbility switch) | Fortune Teller 夜间能力永远不会触发 | ✅ 已修复 |
| 同上 | `game/night.go:476,485` (AbilityInfo Type) | Fortune Teller 结果的 Type 标识错误 | ✅ 已修复 |
| 同上 | `game/setup.go:382` (describeNightAction) | Fortune Teller 夜间行动描述缺失 | ✅ 已修复 |
| 同上 | `engine/engine.go:691` (buildGameContext) | buildGameContext 无法识别 Fortune Teller | ✅ 已修复 |
| Role ID 不匹配：`"scarletwoman"` vs canonical `"scarlet_woman"` | `engine/state.go:550` (CheckWinCondition) | Scarlet Woman 接替恶魔逻辑永远不会触发，影响胜负判定 | ✅ 已修复 |

**根因**：`game/roles.go` 中 `TroubleBrewingRoles` 定义了规范 ID（使用下划线：`fortune_teller`、`scarlet_woman`），但其他文件中部分引用使用了无下划线的旧写法。编译期无法捕获（字符串比较），只有运行时 switch-case 不匹配才暴露。

**面试亮点**：这个 bug 很适合在面试中作为"代码审计发现"的例子——说明了字符串常量散落的风险，以及为什么应该用 const 枚举替代字符串字面量。

---

## 2. Deep Dive Interview Questions（深度面试题）

> 每题附 S 级回答提示，绑定具体代码位置。

### A. 业务建模与架构起手（1-15）

**1) 为什么这个桌游系统不能只用"当前状态表"而要事件流？**
S级提示：围绕 `Store.AppendEvents`、`State.Reduce`、`api.replay` 说明三个核心价值——可追溯（谁在什么时候做了什么）、可复盘（任意时点回放）、可仲裁（事件不可篡改）。补充：在桌游场景中，"昨晚到底发生了什么"是核心玩法，事件流天然匹配。

**2) "确定性"和"创造性"分别放在哪一层？**
S级提示：确定性 = `engine.HandleCommand`（纯函数，输入相同输出必定相同）+ `State.Reduce`；创造性 = `core.Orchestrator.ProcessEvent` → sub-agents → LLM 生成叙事。二者通过"LLM 只消费事件并产出 command，不直接修改 State"这条边界隔离。

**3) 聚合根如何定义？命令入口在哪里？**
S级提示：`RoomActor` + `engine.State` = 房间聚合根。所有写操作统一走 `RoomActor.Dispatch()` → `cmdCh` → `handleCommand()` → `engine.HandleCommand()`。这保证了房间级的事务一致性和全序语义。

**4) 为什么房间级串行比玩家级加锁更贴合业务？**
S级提示：提名、投票、处决天然是房间全序事件（A 提名 B 必须在 C 投票之前），`RoomActor.cmdCh` 自然满足这个语义。玩家级锁会导致死锁风险（多人同时操作同一提名）和顺序不确定性。

**5) 规则判定为何不直接交给 LLM？**
S级提示：LLM 输出不可复算（同一输入不同输出）。判定必须确定性——`engine.HandleCommand` 中 Washerwoman 看到谁、Poisoner 影响谁、Saint 被处决时谁赢，全部硬编码在 `game/night.go`。LLM 只负责"用什么语气告诉玩家结果"。

**6) AutoDM 为什么设计为"事件消费者"而非"状态控制器"？**
S级提示：`AutoDM.OnEvent()` (autodm.go:327) 订阅事件 → 生成响应 → 通过 MCP 工具或 `dispatchCommand` 回写 command → command 再经过引擎验证。如果 AutoDM 直接修改 State，就绕开了事件溯源和一致性保证。

**7) DDD 在这个项目里体现在哪里？短板是什么？**
S级提示：体现——领域对象和不变量集中在 `engine.State/HandleCommand`，不散落在 Controller；房间是天然的聚合边界。短板——没有显式 Aggregate Root 接口、没有 Repository 抽象、没有 Value Object 标注。更准确的说法是"DDD 思想指导了模块划分，但未严格落地战术模式"。

**8) API replay 为什么也做投影裁剪？**
S级提示：`api.go:350-352` 中 `projection.ProjectedState(state, viewer)` 确保离线回放接口与实时 WS 拥有同一套权限模型，避免"通过回放接口泄露恶魔身份"。

**9) 代码中存在两套 Orchestrator，分别是什么？**
S级提示：`agent/orchestrator.go` 是设计态实现（polling-based 7 步循环，使用 zap.Logger），定义了 Sense/BuildContext/Plan/Execute/Observe/Reflect/Persist 完整循环。`agent/core/orchestrator.go` 是运行态实现（event-driven，使用 slog.Logger），被 `AutoDM.NewAutoDM()` 实际构造和调用。前者可视为架构设计文档或未来演进方向。

**10) 房间 context 为什么与 HTTP request context 解耦？**
S级提示：`RoomManager` 持有长生命周期 context (`actorCtx`, room.go:333)，`NewRoomActor` 接收 `loopCtx` 控制 actor 生命周期。如果用 request context，请求超时后 actor 会退出，导致整个房间瘫痪。

**11) 角色可见性隔离如何保证最小披露？**
S级提示：三层裁剪——① `projection.allowed()` 决定事件是否可见（DM 全可见，玩家不能看 bluffs/night_action/poison 等）；② `sanitizePayload()` 删除 `true_role/is_demon/is_minion`；③ `ProjectedState()` 清除全局敏感字段（DemonID, MinionIDs, BluffRoles, NightInfo）。

**12) 如果要支持多剧本（TB/BMR/SnV），第一刀切在哪？**
S级提示：`game/` 层插件化——`roles.go` 和 `night.go` 按 edition 分拆，`engine` 层通过 `State.Edition` 路由到对应角色注册表和夜间结算逻辑。`engine.HandleCommand` 保持协议稳定。

**13) State.Copy() 为什么必须深拷贝？如果浅拷贝会发生什么？**
S级提示：`handleCommand()` 中 `currentState := ra.GetState()` 取当前快照，`nextState := currentState.Copy()` 演进。如果浅拷贝，`nextState.Reduce()` 修改的 Players map 会影响当前 state，导致并发读到脏数据。`state.go:141-201` 深拷贝了 Players/SeatOrder/MinionIDs/BluffRoles/NominationQueue/NightActions/PendingDeaths/Nomination。

**14) 为什么 handleCommand 先算 nextState 再写库，而不是写库后再更新内存？**
S级提示：`room.go:207-238`——先在内存计算 nextState 和 storedEvents（含 seq 赋值），确认一切正确后才调 `AppendEvents` 写入 DB。如果写库后再算 state，DB 写成功但内存更新失败会导致 state 与 DB 不一致。

**15) CheckWinCondition 的检查时机是什么？有没有遗漏场景？**
S级提示：`engine.go` 中在 `HandleCommand` 相关命令处理后调用 `state.CheckWinCondition()` (state.go:543-583)。覆盖了恶魔死亡、Scarlet Woman 接替、仅剩 2 人、Saint 被处决、Mayor 胜利条件。潜在遗漏：Slayer 射击击杀恶魔后的胜利检查（取决于 slayer_shot handler 实现）。

### B. Event Sourcing 与一致性（16-30）

**16) `room_sequences` 的 `FOR UPDATE` 解决了什么问题？**
S级提示：`event_store.go:112` 使用 `SELECT next_seq FROM room_sequences WHERE room_id=? FOR UPDATE` 获取行锁，防止并发 AppendEvents 时 seq 重号或跳号。这是经典的数据库序列号发号器模式。

**17) dedup 记录为什么必须保存完整 CommandResult？**
S级提示：`room.go:218-219`——命中 dedup 时直接返回 `json.Unmarshal(dedup.ResultJSON, &result)` (room.go:174-176)。如果只记录"处理过"而不记录结果，重试请求无法拿到正确的 `AppliedSeqFrom/To`，客户端会认为命令失败。

**18) 快照为什么必须使用"应用后状态"（nextState）？**
S级提示：`room.go:222-223` 判断 `nextState.LastSeq % snapshot == 0` 后用 `engine.MarshalState(nextState)` 生成快照。如果用当前 state（事件应用前），快照 seq 会比实际低一个批次，恢复时重放会重复应用已快照的事件。

**19) `LoadEventsUpTo(to_seq)` 和 `LoadEventsAfter(after_seq, limit)` 的语义差异是什么？**
S级提示：`LoadEventsUpTo` (event_store.go:76-107) 用 `seq<=to_seq` 精确截断，适用于 replay 场景；`LoadEventsAfter` (event_store.go:56-74) 用 `seq>after_seq LIMIT ?` 增量拉取，适用于 WS 补发和 state 恢复。前者保证语义正确，后者保证性能可控。

**20) 什么情况下需要全量重放而不是快照+增量？**
S级提示：审计复盘、历史演进验证、快照怀疑损坏时的校验路径。`api.replay` 就是全量重放的 HTTP 接口。

**21) 事务里同时处理 events + dedup + snapshot 的原因？**
S级提示：`AppendEvents()` (event_store.go:109-151) 在同一个 `WithTx` 中写入事件、dedup、快照。如果分开事务，事件写成功但 dedup 失败会导致重试时重复写入；快照分离会导致快照点与事件不一致。

**22) `write_event` 命令在事件溯源里有什么价值？**
S级提示：将 AutoDM 的治理动作（如 `confirmation.requested`）也纳入不可变事件流（autodm.go:751-770），实现统一审计。没有这个命令，agent 的操作只存在于内存日志中，重启即丢失。

**23) 如何防止"跨房间写错流"？**
S级提示：`handleCommand()` 首行校验 `cmd.RoomID != ra.RoomID` (room.go:164-166)，拒绝不匹配的命令。再加上 `AppendEvents` 的 `FOR UPDATE` 行锁基于 `room_id`，从数据库层再加一道保障。

**24) 命令拒绝和事件写入的边界在哪里？**
S级提示：`engine.HandleCommand` 返回 `events, result, err`。如果 err != nil（如提名条件不满足），`handleCommand()` (room.go:181-183) 直接返回错误，**不写任何事件**。只有 err == nil 时才 AppendEvents。

**25) 如何做"事件 schema 演进"不破坏历史？**
S级提示：event payload 是 `map[string]string`（松散 schema），Reduce 按 key 取值并容忍缺失。新版本可以添加新字段，旧事件仍可被正确 reduce。更严格的做法是在 payload 中加 `"version": "2"` 字段做版本分支。

**26) 断线重连场景如何避免漏事件？**
S级提示：客户端记住最后收到的 `seq`，重连后发送 `subscribe` 消息携带 `last_seq` (ws.go:30-31)。服务端 `handleSubscribe` (ws.go:222) 调用 `LoadEventsAfter(ctx, roomID, payload.LastSeq, 200)` 补发缺失事件，每条都经过 projection 裁剪。

**27) 如果两个客户端同时提交命令，结果是什么？**
S级提示：两个命令都进入 `cmdCh`（缓冲 256），`loop()` 按 FIFO 串行处理。第一个成功后 state 改变，第二个可能被 engine 拒绝（如重复提名同一人）。串行保证不会出现 race condition。

**28) AppendEvents 的 seq 分配策略是什么？**
S级提示：`event_store.go:124-126`——在事务内 `events[i].Seq = current + int64(i)`，然后 `UPDATE room_sequences SET next_seq=?`。一次 command 可能产生多个事件（如 start_game 产生多个 role.assigned），seq 连续递增。

**29) 如何验证 snapshot 正确性？**
S级提示：取同一 room 的 snapshot 后做全量 replay，对比两种方式得到的 State 是否一致（JSON 序列化后对比或关键字段对比）。这是回归测试的核心用例。

**30) LastSeenSeq 字段的设计意图是什么？当前是否实现？**
S级提示：`types.go` 中 `CommandEnvelope.LastSeenSeq` 设计意图是乐观并发控制——客户端声明"我基于 seq=N 的状态发出此命令"，服务端可以检测是否有新事件使该命令无效。**当前未实现**——`handleCommand()` 中未读取或校验此字段。这是一个有意义的演进方向。

### C. Actor / WebSocket / Queue 并发体系（31-45）

**31) 房间 actor 的核心串行化机制是什么？**
S级提示：`RoomActor.loop()` (room.go:120-145) 是单 goroutine 的 `for-select` 循环，从 `cmdCh` 读取命令。所有状态修改只在这个 goroutine 中发生。这是 Go 版本的 Actor 模式——"通过通信共享内存"。

**32) 为什么 GetState() 需要 RWMutex 而不是直接读？**
S级提示：`GetState()` (room.go:311-315) 可能被 HTTP handler / WS handler / broadcast 等多个 goroutine 并发调用，而 `state` 在 `handleCommand` 中被修改 (room.go:235-238)。RWMutex 保证读到一致性状态，且读不互斥。

**33) actor panic 后的恢复流程是什么？**
S级提示：两级 panic recovery：
1. `executeCommand()` (room.go:147-161) 捕获命令级 panic → 返回 error + fatal=true
2. `loop()` (room.go:120-131) 中 fatal 触发 `panic(err)` → defer 捕获 → 调用 `onCrash(roomID)`
3. `handleActorCrash()` (room.go:364-378) 创建新 actor（从 DB 重新 hydrate state），替换 actors map 中的条目

**34) send channel 满时会怎样？**
S级提示：订阅推送 (ws.go:216-219) 使用 `select { case s.send <- b: default: }` 非阻塞投递。满时丢弃消息而非阻塞 broadcast。这避免了慢消费者拖垮整个房间的广播线程。代价是客户端可能漏事件，需要通过 `last_seq` 重连机制补偿。

**35) RabbitMQ 为什么放在 AutoDM 侧而不是命令主链路？**
S级提示：命令主链路（玩家操作 → engine → events）要求低延迟强一致，不能容忍 MQ 延迟和可用性风险。LLM 调用是高延迟、高波动的操作（数秒级），自然适合异步 sidecar。`publishAsyncTask()` (autodm.go:372-390) 失败时降级为同步处理，保证"宁可 LLM 慢点也不能让游戏卡住"。

**36) 队列消费失败的完整重试路径是什么？**
S级提示：`queue.go:187-242`——
1. handler 执行失败 → `task.Retries++`
2. `Retries < MaxRetry` → `Publish(ctx, task)` 重新入队
3. `Retries >= MaxRetry` → 发送到 DLQ (`queueName_dlq`)
4. `msg.Nack(false, false)` 确认消费失败
5. 结果发送到 `resultCh`（非阻塞）

**37) WS 心跳机制如何工作？**
S级提示：`writePump()` (ws.go:148-171) 每 30 秒发送 Ping；`readPump()` (ws.go:114-146) 设置 60 秒读超时 + PongHandler 续期。如果 60 秒内既没有数据也没有 Pong，连接自动关闭。这遵循了 WebSocket 的标准心跳模式。

**38) Token Bucket 限流的配置参数是什么？**
S级提示：`NewTokenBucket(10, 2)` (ws.go:91)——容量 10 令牌，速率 2 令牌/秒。即突发最多 10 个请求，持续速率 2 QPS。超限时返回 `rate_limited` 错误而非断开连接。

**39) DispatchAsync 名称与实际行为是否矛盾？**
S级提示：接口名 `DispatchAsync` (room.go:306-309) 表示"异步来源调用"（如 agent 回调），但内部仍走 `ra.Dispatch(cmd)` 同步等待结果。这是接口语义设计——调用者不关心结果（fire-and-forget），但房间内部仍保持串行一致性。

**40) broadcast 为什么在持久化之后而非之前执行？**
S级提示：`handleCommand()` (room.go:231-240) 先 `AppendEvents`（落库确认），再 `stateMu.Lock` 更新内存状态，最后 `broadcast`。如果先 broadcast，DB 写失败会导致"客户端看到了不存在的事件"——违反 exactly-once 语义。

**41) 为什么 Room 重建不需要通知现有 WS 连接？**
S级提示：`handleActorCrash` (room.go:364-378) 创建新 actor 并替换 map 条目。旧 actor 的 WS 订阅者仍连着旧 actor（已 panic），但读操作（send channel 关闭）会导致连接断开 → 客户端重连 → 订阅新 actor → `last_seq` 补发。自愈设计。

**42) 如果 RabbitMQ 完全不可用，系统行为是什么？**
S级提示：`publishAsyncTask()` (autodm.go:385-388) 返回 false → `OnEvent()` (autodm.go:340) 调用 `ProcessQueuedEvent` 同步处理 → LLM 超时后 fallback。系统完全可用，只是 LLM 响应可能变慢或退化为模板文案。

**43) 广播线程内为什么用 `go ra.autoDM.OnEvent(ctx, ev, state)`？**
S级提示：`broadcast()` (room.go:271) 开新 goroutine 调用 OnEvent，避免 LLM 延迟阻塞后续事件的推送。如果同步调用，一个 LLM 超时（8秒）会卡住所有订阅者的消息推送。

**44) 如果 cmdCh 满了（256 缓冲），Dispatch 会怎样？**
S级提示：`Dispatch()` (room.go:290-294) 有 `select { case ra.cmdCh <- ...: case <-ra.ctx.Done(): }`。如果 cmdCh 满且 ctx 未取消，`Dispatch` 会阻塞直到有空位。这意味着高并发时客户端会感受到延迟上升。可以考虑添加超时或反压机制。

**45) WS subscribe 的 last_seq 是否有并发问题？**
S级提示：`handleSubscribe` (ws.go:196-245) 先注册 subscriber，再通过 `LoadEventsAfter` 补发历史。如果在补发过程中有新事件到达，新事件会通过 subscriber callback 推送。由于补发和实时推送都经过同一个 `s.send` channel 且用同一个 `state` 快照做 projection，不会有一致性问题。但可能有少量事件重复推送（补发窗口内的事件），客户端需基于 seq 去重。

### D. Multi-Agent / LLM Ops / RAG / MCP（46-60）

**46) Orchestrator 的事件路由策略是什么？**
S级提示：`core/orchestrator.go:182-197`——`routeEvent` 按 event.Type 分发：`phase_change` → Narrator；`nomination` → Moderator（验证）；`death` → Narrator（叙事）；`question` → Rules 或 Moderator（关键词检测）；`default` → Moderator。这是一种简单但有效的基于类型的路由。

**47) 如何防止 Agent 自激回路？**
S级提示：`AutoDM.OnEvent()` (autodm.go:331-334) 在入口处检查 `ev.ActorUserID == "autodm" || ev.ActorUserID == "auto-dm"` 并过滤 `public.chat/whisper.sent`。这防止 AutoDM 对自己的发言产生响应，形成无限循环。

**48) LLM 超时治理的完整策略是什么？**
S级提示：
1. `ProcessQueuedEvent` (autodm.go:355-356)：`context.WithTimeout(ctx, a.eventTimeout)` 限制单次事件处理时间（默认 8 秒）
2. RAG 检索独立超时：`1500ms` (autodm.go:457)
3. 超时后 fallback：`defaultMessageForEvent()` (autodm.go:578-593) 返回中文模板消息
4. 异步模式下 RabbitMQ 重试 + DLQ 兜底

**49) 动态上下文注入的完整链路是什么？**
S级提示：
1. `convertEvent()` (autodm.go:392-439)：将引擎事件映射为 agent 内部事件类型
2. `injectRuleContext()` (autodm.go:441-482)：调用 `retriever.Retrieve` 获取相关规则片段
3. `buildRuleQuery()` (autodm.go:484-503)：按事件类型生成模板化查询词
4. 检索结果注入 `event.Data["rule_context"]` 和 `event.Description`
5. 注入后的 event 传给 Orchestrator → Sub-Agent → LLM，LLM 可在上下文中看到规则片段

**50) RAG 查询词为何模板化而不是直接用事件描述？**
S级提示：`buildRuleQuery()` 将事件类型映射为稳定语义查询（如 "nomination and voting rules in Blood on the Clocktower"），而不是用 LLM 生成的事件描述做查询。原因：①避免查询词质量不稳定导致检索漂移；②模板查询可缓存；③桌游规则是结构化的，模板覆盖核心场景即可。

**51) MCP 调用比"直接拼 command"好在哪里？**
S级提示：`Registry.Invoke()` (mcp/registry.go:158-206) 提供了三层保障：①工具注册时声明参数 schema；②调用时 `validateParams` 自动校验；③结果统一返回 `ToolResult`（含 success/error/timestamp），可观测。直接拼 command 缺少参数校验，且调用结果分散在各处日志中。

**52) `request_player_confirmation` 的审计价值是什么？**
S级提示：`autodm.go:703-771`——这个工具执行两个操作：①发 whisper 给玩家；②写 `confirmation.requested` 事件到不可变流。这意味着"AutoDM 什么时候问了谁什么问题"有迹可查，用于事后审计和争议仲裁。

**53) 如何避免 LLM 误判直接影响生死裁决？**
S级提示：生死只能通过 engine 命令触发——`execution.resolved`（投票处决）、`player.died`（夜间死亡）、`slayer_shot`（猎手射击）。LLM/AutoDM 可以调用 `advance_phase` 推进阶段，但不能直接修改玩家的 Alive 状态。即使 LLM 幻觉说"某人死了"，引擎不执行就不生效。

**54) 多模型路由的策略是什么？**
S级提示：`agent/llm/router.go`——Router 按 TaskType 路由到不同模型配置。任务类型包括 reasoning、narration、rules、summarize、quick。每种类型可配置不同的 BaseURL/Model/Temperature/Timeout。例如规则解释用低 temperature 保证准确性，叙事用高 temperature 保证创造性。

**55) 如果 Qdrant 不可用，系统是否可运行？**
S级提示：`injectRuleContext` (autodm.go:441-482) 是增强层——retriever 为 nil 时跳过，检索超时或失败也跳过，不影响主流程。AutoDM 仍可正常响应事件，只是回复中不包含规则引用，幻觉概率可能上升。

**56) MCP schema 校验覆盖了哪些类型？**
S级提示：`validateValue()` (mcp/registry.go:291-364) 递归校验：
- string：enum、minLength、maxLength
- number/integer：minimum、maximum
- boolean：类型检查
- array：items 递归校验
- object：properties 递归校验
缺少 pattern（正则）的实际执行——ParamSchema 定义了 `Pattern` 字段但 `validateValue` 未检查。

**57) Agent memory 的短期记忆如何工作？**
S级提示：`agent/memory/manager.go`——Manager 维护固定大小的环形缓冲区，`AddEvent()` 按 room 存储事件描述，`Recent()` 返回最近 N 条，`GetContext()` 生成格式化上下文串。内存在进程重启后丢失，长期记忆需要外部存储。

**58) 两套 Orchestrator 如何在面试中表述？**
S级提示：诚实说法——`agent/orchestrator.go` 是设计阶段的完整控制循环（7 步），体现了 Agent 系统的理论架构。后来发现事件驱动模式更适合实时游戏场景，于是实现了 `agent/core/orchestrator.go`（event-driven）。前者保留作为架构参考和未来增强方向（如需要 ticker-based 主动巡检时可启用）。

**59) AutoDM sendMessage 的双路径是什么？**
S级提示：`sendMessage()` (autodm.go:530-572)——先尝试 MCP `send_public_message` 工具（经过 schema 校验），失败后降级为直接构造 `CommandEnvelope` dispatch。这是"MCP 优先，直连兜底"的双路径设计。

**60) 当前系统最大的技术债是什么？下一步最值得做什么？**
S级提示：
- **技术债**：`LastSeenSeq` 乐观并发未实现、Pattern 校验未执行、双 Orchestrator 冗余、`agent/memory.go` 的 SimpleEmbedder 是伪实现
- **下一步**：① Agent FSM 显式化（状态转换表替代 switch-case）；② `LastSeenSeq` 冲突检测；③ 跨实例 room 路由（一致性哈希）；④ 删除或整合 `agent/orchestrator.go`

### E. 测试与可观测（61-70）

**61) 如何测试事件流重建正确性？**
S级提示：同一房间做 full replay 与 snapshot+tail replay，对比最终 State。

**62) 如何测试幂等？**
S级提示：同一 `idempotency_key` 多次 Dispatch，验证 `CommandResult` 一致且 events 表无重复。

**63) 如何测试投影权限？**
S级提示：构造 DM/玩家/旁观三类 viewer，覆盖 `role.assigned`、`whisper.sent`、夜晚事件。

**64) 如何测试 actor crash recovery？**
S级提示：注入一个会 panic 的命令 → 验证 `onCrash` 被调用 → 新 actor 的 state 与 DB 一致。

**65) 如何测试 LLM timeout fallback？**
S级提示：mock LLM 返回超时 → 验证 `defaultMessageForEvent` 被调用 → 验证消息成功发送。

**66) 上线后优先监控哪些指标？**
S级提示：command latency P95/P99、dedup hit rate、WS resync event count、queue retry/DLQ count、AutoDM timeout rate、active connections。

**67) 怎么证明异步旁路降低了主链路抖动？**
S级提示：对比启用/禁用 queue 时 command P95/P99 与 WS 心跳超时率。

**68) 如何做 MCP 工具的集成测试？**
S级提示：构造 ToolCall → 调用 Registry.Invoke → 验证参数校验 + handler 执行 + 结果格式。

**69) 如何测试 RAG 注入质量？**
S级提示：用已知规则文档 → 触发特定事件 → 验证 `event.Data["rule_context"]` 包含相关片段。

**70) 如何做端到端的游戏回合测试？**
S级提示：脚本化整局游戏（join → start → night → day → nominate → vote → execute → check win），验证每一步的 events/state/projection。

---

## 3. 3 分钟"技术难点与设计决策"独白

> 面试场景：面试官说"请花 3 分钟聊一个你做过的技术难点"。

### 核心矛盾

这个项目最核心的矛盾，是把一个**必须确定性**的规则系统，与一个**概率性**的 LLM 生成模型放在同一个实时多人服务里。桌游的规则裁判不允许任何偏差——中毒后能力失效、恶魔死亡好人胜利，这些必须像数据库事务一样可靠。但同时，AI 主持人需要生成自然语言叙事、回答规则问题、营造氛围，这些天然是概率性的。

### 第一层：确定性内核

解决方案的第一层是**Event Sourcing + 纯函数引擎**。所有游戏状态变化都由 `engine.HandleCommand` 驱动——输入是命令和当前状态，输出是事件列表，完全无副作用。事件落入 append-only 的 MySQL 表，`State.Reduce` 按事件类型演进状态。这使得任何时刻的状态都可以从事件流精确重建，用于复盘、审计、争议仲裁。

### 第二层：并发一致性

第二层是**房间级 Actor 串行化**。每个房间一个 goroutine，所有命令通过 channel 排队，天然满足全序语义。投票、提名这类操作在房间内必须有严格顺序，Actor 模式比多锁更贴合业务。幂等通过 `idempotency_key` + 数据库去重保证，断线重连通过 `last_seq` 增量补发，事件回放通过 `to_seq` 精确重建。

### 第三层：LLM 控制边界

第三层是**MCP 工具边界约束 LLM**。LLM 不直接修改游戏状态，只能通过注册的 MCP 工具发出受控动作——发消息、推进阶段、写审计事件。每个工具调用必须通过 JSON Schema 校验。慢 LLM 调用旁路到 RabbitMQ 异步处理，超时或失败降级为模板文案，保证游戏流程永不阻塞。RAG 在事件处理前注入相关规则片段，降低幻觉概率但不阻塞主路径。

### 技术取舍

放弃了"单表当前态覆盖"——不可追溯、不可复盘。放弃了"LLM 同步直连"——一个 8 秒超时会卡住所有玩家。选择 Event Sourcing + Actor + MCP + 异步 sidecar，核心目标是：**规则必须像数据库事务一样可靠，体验可以像 AI 助手一样自然**。

---

## 4. 简历 Reality Check（现实校验）

### ✅ 可直接保留的强表述

| 声明 | 代码支撑 |
|------|----------|
| Event Sourcing（append-only + snapshot + last_seq + replay） | `store/event_store.go`, `room/room.go`, `api.go:replay` |
| 房间级 Actor 串行 + 幂等去重 + crash recovery | `room/room.go:120-378` |
| 可见性投影三层裁剪（allowed → sanitize → projectedState） | `projection/projection.go` |
| MCP 工具调用 + JSON Schema 校验 + 审计事件回写 | `mcp/registry.go`, `autodm.go:605-928` |
| RAG 动态上下文注入（Qdrant + OpenAI Embedding） | `rag/`, `autodm.go:441-503` |
| LLM 旁路异步（RabbitMQ + DLQ + retry）+ timeout fallback | `queue/queue.go`, `autodm.go:347-390` |
| Multi-Agent 架构（Orchestrator + 5 Specialists） | `agent/core/orchestrator.go`, `agent/subagent/` |
| TB 全角色夜间能力实现（含中毒/醉酒失灵逻辑） | `game/night.go`, `game/roles.go` |

### ⚠️ 建议收敛的措辞

| 原始表述 | 代码现实 | 建议修改 |
|----------|----------|----------|
| "完全无锁 Actor" | `stateMu`/`subsMu` 是 RWMutex | "房间级 Actor 串行为主，读视图使用轻量读写锁保护" |
| "完整 FSM 状态编排" | Phase/SubPhase 枚举转换散落在 Reduce() switch-case | "规则层 FSM 已稳定运行（7 Phase × 5 SubPhase），Agent 层显式状态机作为演进方向" |
| "DDD 全栈落地" | 无显式 Aggregate Root/Repository/Value Object | "DDD 思想指导模块划分，聚合边界明确（Room = 聚合根），战术模式持续强化" |
| "基于 EINO 框架" | 代码中无 EINO 引用，全部自建 | **删除此声明**，改为"自建 Multi-Agent 框架" |
| "7 步 Agent 控制循环" | 实际运行的是 event-driven 路由 | "设计了 Sense→Plan→Execute→Reflect 控制循环，当前以 event-driven 模式运行" |

### ❌ 必须修改的声明

1. **EINO 框架**：代码中无任何 EINO 引用。整个 agent 系统（LLM routing、memory management、tool registry、sub-agents）全部自建。建议改为"自建 Multi-Agent 框架，支持 LLM 路由、短期记忆、工具注册"。

2. **DDD 严格落地**：当前实现更接近"领域模型集中"而非"战术 DDD"。没有 Repository 接口（直接用 `store.Store`）、没有 Value Object 标注、没有 Domain Service 抽象。

### 💡 可追加的亮点描述

- "通过将 LLM 不确定性限制在 MCP 工具边界内，并将关键动作全部事件化，形成'可创造但可审计'的 AI 游戏主持架构。"
- "确定性引擎（纯函数 HandleCommand + Event Sourcing）与概率性 Agent（LLM + RAG）解耦，保证规则可靠性与体验灵活性共存。"
- "自建 Multi-Agent 框架，5 个 Specialist Agent 各司其职，Orchestrator 按事件类型路由，支持多模型路由和短期记忆管理。"

### 📊 量化数据建议

面试中用数据说话更有说服力：
- **事件类型**：25+ 种（player.joined, role.assigned, phase.*, nomination.*, vote.cast, execution.*, ability.*, ...）
- **MCP 工具**：6 个核心工具 + 11 个扩展工具
- **TB 角色**：13 Townsfolk + 4 Outsider + 4 Minion + 1 Demon = 22 角色全覆盖
- **状态机**：7 Phase × 5 SubPhase
- **Actor 恢复**：两级 panic recovery（命令级 + actor 级）
- **Projection 裁剪规则**：6 类敏感事件过滤 + 3 类字段清理

---

## 5. 附录：关键代码路径速查

### 命令处理主链路
```
WS handleCommand (ws.go:247)
  → RoomActor.Dispatch (room.go:288)
    → cmdCh → loop → executeCommand (room.go:147)
      → handleCommand (room.go:163)
        → dedup check (room.go:168)
        → engine.HandleCommand (engine.go)
        → AppendEvents (event_store.go:109) [事务: events + dedup + snapshot]
        → state update (room.go:235)
        → broadcast (room.go:240)
          → projection.Project per subscriber (room.go:261-267)
          → autoDM.OnEvent (room.go:270-272)
```

### AutoDM 响应链路
```
AutoDM.OnEvent (autodm.go:327)
  → updateGameStateFromEngineState (autodm.go:947)
  → publishAsyncTask (autodm.go:372) [RabbitMQ]
    ↓ (async worker dequeue)
  → ProcessQueuedEvent (autodm.go:347)
    → convertEvent (autodm.go:392)
    → injectRuleContext (autodm.go:441) [RAG: Qdrant query → inject snippets]
    → ProcessEvent (autodm.go:214)
      → core.Orchestrator.ProcessEvent (core/orchestrator.go:165)
        → routeEvent (core/orchestrator.go:182)
          → narrator/moderator/rules sub-agent → LLM call
    → sendMessage (autodm.go:530)
      → MCP send_public_message (autodm.go:538-553)
      → fallback: dispatchCommand (autodm.go:559-571)
```

### 状态恢复链路
```
NewRoomActor → loadState (room.go:80)
  → GetLatestSnapshot (event_store.go:39)
  → UnmarshalState (state.go:487)
  → LoadEventsAfter (event_store.go:56)
  → State.Reduce per event (state.go:203)
```
