# Blood on the Clocktower Auto-DM — AI 自动主持的血染钟楼多人在线游戏

## 技术栈
- **前端**: Vue 2.6.12 + Vuex 3 + Vue I18n 8 + Vue CLI 5, SCSS, FontAwesome
- **后端**: Go 1.25 + Chi (HTTP) + Gorilla WebSocket
- **数据库**: MySQL 8.0 (事件存储) + Redis 7 (缓存)
- **消息队列**: RabbitMQ 3.12 (异步 Agent 任务)
- **AI**: Gemini / OpenAI / Deepseek (多模型路由), Qdrant 向量库 (规则 RAG)
- **可观测性**: Prometheus + Grafana + OpenTelemetry + Zap 日志
- **认证**: JWT (HS256, 24h TTL)
- **通信协议**: REST (状态同步) + WebSocket (实时事件推送)

## 目录结构
- `backend/` → Go 后端服务
  - `cmd/server/` → 入口 main.go，初始化所有依赖并启动 HTTP 服务
  - `internal/engine/` → 游戏状态机，命令分发，胜负判定 (核心，1095 行)
  - `internal/game/` → 角色定义、夜晚行动解析、游戏初始化
  - `internal/agent/` → Auto-DM AI 系统：编排器、子代理 (主持/叙事/规则/摘要/玩家建模)
  - `internal/api/` → HTTP 路由 + 命令处理，Swagger 文档
  - `internal/realtime/` → WebSocket 服务器，订阅/广播，令牌桶限流
  - `internal/projection/` → 事件可见性过滤 (不同玩家看到不同信息)
  - `internal/store/` → MySQL 事件存储 + 快照 + 幂等去重
  - `internal/auth/` → JWT 生成/验证 + bcrypt 密码
  - `internal/room/` → 房间管理，Actor 模型 (每房间独立命令队列)
  - `internal/queue/` → RabbitMQ 异步任务 (autodm_event)
  - `internal/rag/` → Qdrant 向量检索，规则语义搜索
  - `internal/bot/` → 测试用 Bot 玩家
  - `internal/config/` → 环境变量加载
  - `internal/observability/` → Prometheus 指标 + OTel 追踪
  - `db/migrations/` → SQL 建表迁移
  - `loadtest/` → 压测工具与场景脚本
- `frontend/` → Vue 2 单页应用
  - `src/components/` → 24 个 Vue 组件 (屏幕/覆盖层/面板)
  - `src/store/modules/` → 8 个 Vuex 模块: game, players, chat, vote, night, timeline, ui, annotations
  - `src/store/plugins/` → persistence (localStorage) + websocket (事件监听/命令发送)
  - `src/services/` → ApiService (REST+WS), SoundService
  - `src/i18n/` → 中英双语 (en.json, zh.json)，浏览器自动检测
  - `src/assets/` → 角色图标 (154+), 字体 (Papyrus/PiratesBay), 音效
  - `src/*.json` → 角色数据 (roles.json 65KB), 剧本 (editions.json), 传奇 (fabled.json)
- `docs/` → 需求文档 + 规则文档 (rules/ 子目录供 RAG 索引)
- `townsquare/` → 旧版参考实现 (独立 git 子项目，不参与构建)

## 架构决策
1. **事件溯源 (Event Sourcing)**: 所有游戏状态变更为不可变事件存入 MySQL；状态 = 初始态 + Reduce(事件)；每 50 事件自动快照加速加载
2. **Actor 模型**: 每个房间一个 RoomActor，命令队列串行处理，消除并发竞态
3. **可见性投影 (Projection)**: 服务端单一事实源，按玩家角色过滤事件 (如当前角色只能看到自己发动技能而看不到其他角色发送技能、无法看见其他玩家角色身份)，防信息泄漏
4. **多代理 AI 系统**: Orchestrator 协调 5 个子代理 (Moderator/Narrator/Rules/Summarizer/PlayerModeler)，通过 RabbitMQ 异步处理，不阻塞游戏流程
5. **无 Vue Router**: 画面路由由 Vuex `ui.screen` 状态驱动 (home/lobby/game/end)，简化移动端导航

## 模块依赖关系
- 前端 `websocket.js` → 后端 `/ws` (WebSocket 实时事件推送 + 命令发送)
- 前端 `ApiService.js` → 后端 `/v1/*` (REST: 认证、建房、加入、状态同步)
- 后端 `api.go` → `room.RoomManager` → `engine.HandleCommand()` (命令分发)
- `engine` → `game` (角色定义、夜晚解析、胜负规则)
- `RoomActor` → `store.EventStore` (事件持久化) → `projection.Project()` (可见性过滤) → `WSServer.Broadcast()` (推送给订阅者)
- `RoomActor` → `queue.TaskQueue` (异步发布 autodm_event) → `agent.AutoDM` (AI 处理)
- `agent.AutoDM` → `agent/llm` (LLM 调用) + `rag.Retriever` (规则检索)

## 代码红线

### Go 后端
- 单文件 ≤ 500 行（engine.go 1095 行为历史遗留，**禁止新增函数**，新功能拆到独立文件）
- 单函数 ≤ 50 行（含注释和空行）
- 嵌套 ≤ 3 层（if/for/switch/闭包各算一层）
- 单函数分支 ≤ 3 个 if/else（超过用 switch 或 early return 重构）
- 函数参数 ≤ 4 个（超过用 struct 或 functional options）
- 接口方法 ≤ 5 个
- error 必须处理，禁止 `_ = err`
- 错误信息格式：`fmt.Errorf("模块.函数: %w", err)`，必须带上下文
- 禁止 panic（init() 配置校验除外）
- goroutine 必须有 context 控制生命周期，必须 `defer recover`
- channel 必须有 close 机制
- 函数名 = 动词+名词（SendMessage, ValidateVote），禁止单独的 Process/Handle/Do/Run
- Bool 变量用 is/has/can/should 前缀

### Vue 前端
- 单组件 ≤ 400 行（template + script + style 总计）
- JS 单函数 ≤ 50 行
- 单个 Vuex module 文件 ≤ 300 行
- 组件 props ≤ 6 个（超过用 provide/inject 或拆组件）
- computed 属性不做副作用（纯计算，不改数据）
- template 中表达式嵌套 ≤ 2 层（超过抽成 computed）
- 禁止在 mounted/created 里直接调 ApiService，必须通过 Vuex action
- SCSS 颜色值禁止硬编码，必须用 vars.scss 变量
- 组件间通信：父子用 props/emit，跨层用 Vuex，禁止 event bus

### 架构边界
- engine 包禁止 import agent 包（状态机不知道 AI 的存在）
- agent 包禁止直接修改 engine.State，必须通过 CommandEnvelope
- projection 包禁止 import store 包（只做读过滤，不写存储）
- realtime 包禁止 import engine 包（WebSocket 层不包含游戏逻辑）
- 前端 components 禁止直接 import ApiService，必须通过 Vuex action
- Vuex modules 之间禁止直接 import，跨模块用 rootGetters/rootActions
- 前端禁止在组件中直接操作 sessionStorage/localStorage，必须通过 StorageService

### 项目约定
- ESLint 严格模式: no-unused-vars 违反会导致构建失败
- Vue 组件名允许单词（已关闭 `vue/multi-word-component-names`）
- SCSS 变量统一定义在 `vars.scss`（$townsfolk, $outsider, $minion, $demon, $fabled, $traveler）
- 前端认证使用 sessionStorage（每个标签页独立身份，支持多玩家测试）
- 后端所有命令 payload 为 `map[string]string`；数组需 JSON 序列化为字符串
- i18n: 用 `$te(key)` 检查翻译是否存在，回退到英文数据
- 座位号后端 1-indexed；seatIndex = -1 表示未入座

## 工作流
- **启动后端**: `cd backend && make docker-up && make run-env`（需先配置 `.env`）
- **启动前端**: `cd frontend && npm install && npm run serve`（端口 8081）
- **跑测试**: `cd backend && make test`
- **新功能**: 先 plan mode 出方案 → 审核 → 执行
- **改 bug**: 只加载目标模块文档，定位最小改动范围
- **每次代码变更后检查相关 CLAUDE.md 是否需要更新**

## Plan 管理规则

### 生成规则
- 每次进入 plan mode 做新功能时，方案确认后必须保存为 `/Users/qingchang/Blood-on-the-Clocktower-auto-dm/.claude/plans/YYYY-MM-DD-NNN-功能名.md`（如 `2026-02-25-001-fix-game-flow.md`）
- NNN 为当日递增编号（001 起），同日多个 plan 依序递增
- Plan 文件必须包含 checklist，每个实现步骤用 `- [ ]` 标记
- 最后一步永远是 `- [ ] 回环检查：更新所有受影响的 CLAUDE.md 和文件头注释`
- Plan 文件末尾必须有状态行，格式：`## 状态：🔄 进行中 - 当前在第 X 步`
- **Plan mode 路径同步**：plan mode 系统会将文件写入 `~/.claude/plans/`（随机文件名），退出 plan mode 后必须立即将该文件复制到项目的 `.claude/plans/` 目录并按 `YYYY-MM-DD-NNN-功能名.md` 规范重命名，然后删除全局目录下的原文件。后续所有编辑只操作项目目录下的副本。

### 执行规则
- 每完成一个步骤，立即将对应行改为 `- [x]` 并更新状态行
- 不要攒着一起改，完成一步改一步——这是会话中断后恢复的唯一依据

### 会话恢复规则
- 如果 `.claude/plans/` 中存在状态为"进行中"的 plan 文件，先读取该文件，从第一个未完成的 `- [ ]` 步骤继续执行
- 恢复时先汇报：当前在执行哪个 plan、已完成哪些步骤、接下来要做什么
