# INTERVIEW PREP - Blood on the Clocktower Auto-DM

> 更新时间：2026-02-05  
> 范围：`backend/cmd`, `backend/internal`（重点 `agent/engine/room/store/realtime/projection/mcp/rag`）

---

## 1. 修复后亮点（由原问题转亮点）

- **AutoDM 主链路完整可运行**：服务启动阶段完成 `SetDispatcher + Start + Stop`，并修复 AutoDM 发言命令为引擎可识别的 `public_chat`，避免“配置开启但无行为”的假启用。
- **LLM 旁路异步化闭环**：`AutoDM.OnEvent -> publishAsyncTask -> queue handler -> ProcessQueuedEvent` 打通；失败时自动降级为同步处理并使用 fallback 文案，主游戏流程持续推进。
- **事件溯源一致性增强**：`RoomActor.handleCommand` 先计算 `CommandResult` 再写 dedup；快照改为基于 `nextState` 触发，避免快照与事件序列错位。
- **精确回放能力完善**：新增 `Store.LoadEventsUpTo`，`replay` 接口按 `seq<=to_seq` 精确重建，不再用 LIMIT 近似。
- **Actor 稳定性升级**：房间 actor 生命周期从 request context 解耦到 manager-level context；命令路径 panic 会返回错误并触发 actor 重建流程。
- **并发安全增强**：`RoomActor.GetState` 改为读锁保护；`Orchestrator.ProcessEvent` 在锁内复制关键字段后写 memory，减少竞态窗口。
- **可见性隔离补齐**：WS 增量补发统一走 `projection.Project`；非 DM 敏感事件与敏感状态字段（恶魔/爪牙/真身份/夜晚信息）持续裁剪。
- **MCP 工具边界完整化**：在 AutoDM 运行链路内注册并启用 `send_public_message`、`send_private_message`、`request_player_confirmation`、`toggle_voting`、`advance_phase`、`write_event`，全部经过 schema 校验。
- **可审计写事件能力补全**：引擎新增 `write_event` 命令（`engine.HandleCommand -> handleWriteEvent`），支持 DM/AutoDM 以标准化方式写入不可变事件流。
- **重连与幂等体验改进**：WS command 缺失 `command_id/idempotency_key` 时服务端自动补齐 UUID，减少客户端接入差异导致的幂等失效。
- **RAG 动态注入落地主路径**：`injectRuleContext` 在事件处理时注入规则片段，结合 phase/nomination/vote/death 模板查询，降低长规则场景幻觉概率。

---

## 2. Deep Dive Questions（单点集中，循序渐进，60题）

> 说明：每题给出 **S级回答提示**，并绑定代码中的函数/结构体名称，便于面试时展开。

### A. 业务建模与架构起手（1-12）

1) 为什么这个桌游系统不能只用“当前状态表”而要事件流？  
S级回答提示：围绕 `Store.AppendEvents`、`engine.State.Reduce`、`api.replay` 说明可追溯、可复盘、可争议仲裁。

2) 这个系统里“确定性”和“创造性”分别放在哪一层？  
S级回答提示：确定性在 `engine.HandleCommand`；创造性在 `agent/core.Orchestrator` 与 `AutoDM.ProcessQueuedEvent`。

3) 聚合根如何定义？  
S级回答提示：`RoomActor + engine.State` 作为房间聚合根，命令入口统一 `RoomActor.handleCommand`。

4) 为什么房间级串行比玩家级加锁更贴合业务？  
S级回答提示：提名/投票天然是房间全序语义，`RoomActor.cmdCh` 与 `Dispatch` 更符合一致性边界。

5) 规则判定为何不直接交给 LLM？  
S级回答提示：判定必须可复算，LLM 放在叙事/解释；真实状态变更由 `engine` 命令处理。

6) 为什么将 AutoDM 设计为“事件消费者”而非“直接状态控制器”？  
S级回答提示：`AutoDM.OnEvent` 订阅事件并回写 command，避免绕开事件溯源。

7) 这个项目的 DDD 体现在哪里？  
S级回答提示：领域对象与不变量集中在 `engine.State`、`engine.HandleCommand`，不是 Controller 中散落逻辑。

8) 为什么 API replay 也做投影裁剪？  
S级回答提示：`projection.ProjectedState` 保持与实时 WS 同权限模型，避免“离线接口泄露”。

9) 如何理解“简历中的 Orchestrator + Specialists”在代码中的落点？  
S级回答提示：`core.Orchestrator` 路由到 `Moderator/Narrator/Rules/Summarizer/PlayerModeler` 子 agent。

10) 为什么要把房间上下文与 HTTP 请求上下文解耦？  
S级回答提示：`RoomManager` 持有长生命周期 context，避免 request 结束导致 actor 退出。

11) 角色可见性隔离如何保证最小披露？  
S级回答提示：`projection.allowed` + `projection.sanitizePayload` + `projection.ProjectedState` 三层裁剪。

12) 如果要支持多剧本（TB/BMR/SnV），第一刀切在哪？  
S级回答提示：`game` 层角色定义与夜间行动解析插件化，`engine` 维持协议稳定。

### B. Event Sourcing 与一致性（13-26）

13) `room_sequences` 的 `FOR UPDATE` 解决了什么问题？  
S级回答提示：防止并发 append 时 seq 重号/乱序，保证单房间严格递增。

14) dedup 记录为什么必须保存完整 `CommandResult`？  
S级回答提示：`GetDedupRecord` 命中后应返回与首次执行一致结果，避免客户端重试分叉。

15) 之前 dedup 的时机 bug 是什么，修复点是什么？  
S级回答提示：`RoomActor.handleCommand` 先算 `AppliedSeqFrom/To` 并写 `ResultJSON`，再事务落库。

16) 快照为什么必须使用“应用后状态”？  
S级回答提示：`nextState` 才对应本批事件后的真值，避免 snapshot 与事件尾部不一致。

17) 为什么 `to_seq` 不能靠 LIMIT 近似？  
S级回答提示：LIMIT 不等价于 seq 边界；`LoadEventsUpTo` 用 `seq<=to_seq` 语义正确。

18) 状态重建路径如何组合？  
S级回答提示：`GetLatestSnapshot` + `LoadEventsAfter`（热路径）或 `LoadEventsUpTo`（回放路径）。

19) 什么情况下需要全量重放而不是快照+增量？  
S级回答提示：审计复盘、历史演进验证、snapshot 怀疑损坏时的校验路径。

20) 如何做“事件 schema 演进”不破坏历史？  
S级回答提示：event payload 加版本字段，`Reduce` 做版本兼容分支。

21) 如何在恢复时验证状态正确性？  
S级回答提示：状态哈希（players/phase/seq）对比 + 随机抽样 replay 校验。

22) 断线重连场景如何避免漏事件？  
S级回答提示：客户端携带 `last_seq`，服务端 `LoadEventsAfter` 回补并更新本地游标。

23) 命令拒绝与事件写入的边界在哪里？  
S级回答提示：拒绝发生在 `engine.HandleCommand` 前，不写事件；接受后统一 append。

24) 事务里为什么要同时处理 events + dedup + snapshot？  
S级回答提示：保证命令结果、事件序列、快照点原子一致。

25) `write_event` 命令在事件溯源里有什么价值？  
S级回答提示：将审核/确认/治理类动作也纳入同一不可变流，实现统一审计面。

26) 如何防止“跨房间写错流”？  
S级回答提示：`RoomActor.handleCommand` 首先校验 `cmd.RoomID == actor.RoomID`。

### C. Actor / WS / Queue 并发体系（27-40）

27) 房间 actor 的核心串行化机制是什么？  
S级回答提示：`RoomActor.loop` 单 goroutine 消费 `cmdCh`，命令天然全序。

28) 读状态为什么也需要并发保护？  
S级回答提示：`GetState` 与命令处理并发，读锁避免 data race。

29) actor panic 后如何恢复？  
S级回答提示：`executeCommand` 捕获命令 panic -> 返回错误 -> 外层 panic -> `onCrash` 重建 actor。

30) Room 重建如何保持数据正确？  
S级回答提示：重建时调用 `loadState`，按 snapshot + events 重新 hydrate。

31) 为什么 WS 重连补发必须复用 projection？  
S级回答提示：`handleSubscribe` 重放事件调用 `projection.Project`，避免补发通道泄露敏感字段。

32) WS command 缺少幂等字段如何处理？  
S级回答提示：`handleCommand` 自动生成 UUID 填充 `command_id/idempotency_key`。

33) 如何解释 `DispatchAsync` 名称与行为？  
S级回答提示：接口语义是“异步来源调用”；房间内部仍串行落地，保证顺序一致。

34) send channel 满时当前策略是什么？  
S级回答提示：订阅推送使用非阻塞投递，避免慢消费者拖垮房间广播。

35) RabbitMQ 为什么放在 AutoDM 事件侧而不是命令主链路？  
S级回答提示：命令主链路要低延迟强一致，LLM 属于高波动耗时任务。

36) 队列消费失败的重试策略是什么？  
S级回答提示：`queue.processMessage` 按 `Retries/MaxRetry` 回投，超限进 DLQ。

37) 为什么 queue handler 要先注册后启动？  
S级回答提示：避免“无 handler 直接 Nack 丢任务”。

38) WS 与 queue 的边界如何约束？  
S级回答提示：queue 只产出 command/event，不直接修改 room 内存状态。

39) 房间上下文取消后，Dispatch 如何反馈？  
S级回答提示：`Dispatch` 监听 `ra.ctx.Done()` 返回 `room actor stopped`，防止无限阻塞。

40) 为什么不直接在广播线程内同步调用 LLM？  
S级回答提示：广播线程要快返回，否则会放大房间内所有操作延迟。

### D. Multi-Agent / FSM / MCP / RAG / LLM Ops（41-54）

41) 项目里的 FSM 在哪一层最关键？  
S级回答提示：规则 FSM 在 `engine` phase/subphase；Agent 控流在 `core.Orchestrator` 事件路由。

42) 如何防止 Agent 自激回路（自己回复自己）？  
S级回答提示：`AutoDM.OnEvent` 过滤 `autodm` 产生的 `public.chat/whisper.sent`。

43) LLM 超时如何治理？  
S级回答提示：`ProcessQueuedEvent` 使用 `context.WithTimeout`，失败使用 fallback 文案。

44) 动态上下文注入具体是如何做的？  
S级回答提示：`injectRuleContext` 调 `retriever.Retrieve`，把规则片段拼入 event 描述。

45) RAG 查询词为何按事件类型模板化？  
S级回答提示：`buildRuleQuery` 将 phase/nomination/vote/death 映射为稳定语义查询。

46) MCP 调用为什么比“直接拼 command”更可控？  
S级回答提示：`Registry.Invoke` 先过 schema 校验，再执行 handler，失败可观测。

47) 关键 MCP 工具覆盖了哪些治理动作？  
S级回答提示：`send_public_message/send_private_message/request_player_confirmation/toggle_voting/advance_phase/write_event`。

48) `request_player_confirmation` 的审计价值是什么？  
S级回答提示：工具既发 whisper，又写 `confirmation.requested` 事件，实现沟通与审计一致。

49) `toggle_voting` 为什么映射到阶段推进？  
S级回答提示：在现有引擎语义里“投票开关”通过 `advance_phase` 到 `nomination/day` 实现。

50) 为什么要新增 `write_event` 到引擎而不是只在 agent 内存里记日志？  
S级回答提示：内存日志不可重放；事件流可审计、可恢复、可统一回放。

51) 如何避免 LLM 误判直接影响生死裁决？  
S级回答提示：生死只允许 `engine` 命令触发，LLM 仅生成叙事或建议。

52) 如何做多模型路由？  
S级回答提示：`llm.Router` 按任务类别路由（规则解释、叙事、总结）并配置不同模型/超时。

53) 若 RAG 不可用，系统是否可运行？  
S级回答提示：`injectRuleContext` 是增强层，检索失败不影响命令主路径。

54) MCP schema 当前覆盖到哪种深度？  
S级回答提示：`registry.validateParams` 覆盖 required/type/enum/minmax/object/array 校验。

### E. 测试、可观测、演进（55-60）

55) 如何测试“事件流重建正确性”？  
S级回答提示：同一房间做 `full replay` 与 `snapshot+tail replay` 状态比对。

56) 如何测试幂等与重复提交安全？  
S级回答提示：相同 `idempotency_key` 多次提交，验证 `seq` 与 `CommandResult` 不漂移。

57) 如何测试投影权限隔离？  
S级回答提示：构造 DM/玩家/旁观三类 viewer，覆盖 `role.assigned`、`whisper.sent`、夜晚事件。

58) 上线后优先监控哪些指标？  
S级回答提示：房间 command latency、dedup hit、WS resync count、queue retries/DLQ、AutoDM timeout 率。

59) 怎么证明旁路异步确实降低了主链路抖动？  
S级回答提示：对比启用/禁用 queue 时 command P95、P99 与 WS 心跳超时率。

60) 下一阶段最值得继续投入的工程点是什么？  
S级回答提示：Agent 控流显式 FSM、`LastSeenSeq` 冲突策略、跨实例 room 路由与一致性哈希。

---

## 3. 3 分钟“技术难点与设计决策”独白

这个项目最核心的矛盾，是把一个必须确定性的规则系统，与一个概率性的生成模型放在同一个实时服务里。

第一层设计是确定性内核。提名、投票、处决、夜晚结算全部由 `engine.HandleCommand` 驱动，任何状态变化都先落为事件，再由 `State.Reduce` 演进。这样每一步都可回放、可审计、可复盘。

第二层设计是并发一致性。房间级 Actor 用单 goroutine 串行处理命令，天然满足房间内全序语义，避免玩家级多锁带来的死锁与顺序分叉。幂等通过 `idempotency_key` 落库去重，断线重连通过 `last_seq` 补发，调试回放通过 `to_seq` 精确重建。

第三层设计是 LLM 控制边界。LLM 不直接改状态，只消费事件并产出受控动作；慢调用旁路到 RabbitMQ，超时或失败降级到 fallback 文案，保证流程不断。工具调用统一走 MCP 注册表，参数必须通过 schema 校验，关键动作继续写入事件流，保证可追踪。

方案取舍上，放弃了“单表当前态覆盖”和“同步直连 LLM”，选择 Event Sourcing + Actor + 异步 sidecar。前者解决可追溯与一致性，后者保证实时链路稳定。最终目标是：规则必须像数据库事务一样可靠，体验可以像 AI 助手一样自然。

---

## 4. 简历 Reality Check（修订建议）

- 可直接保留的强表述：
  - 事件溯源（append-only、snapshot、incremental sync、replay）
  - 房间级 Actor 串行 + 幂等去重 + 可见性投影
  - LLM 旁路异步化 + timeout fallback
  - MCP 工具调用 schema 校验 + 可审计事件回写
  - RAG 动态上下文注入

- 建议收敛的措辞：
  - “完全无锁 Actor”建议改为“房间级 Actor 串行为主，必要位置采用轻量锁保护共享读视图”。
  - “Agent FSM 完整编排”建议改为“规则层 FSM 已稳定，Agent 层显式状态机正在持续强化”。
  - “DDD 全栈落地”建议改为“领域模型与聚合边界已明确，应用服务与跨上下文边界持续演进”。

- 可追加的一句话亮点：
  - “通过将不确定性限制在 MCP 工具边界内，并将关键动作全部事件化，形成‘可创造但可审计’的 AI 游戏主持架构。”

---

## 5. 本轮回归验证（简版）

- `go test ./...` 通过。
- `go test -race ./internal/agent ./internal/room ./internal/realtime ./internal/projection ./internal/engine` 通过。
- `go build ./cmd/server` 通过。

