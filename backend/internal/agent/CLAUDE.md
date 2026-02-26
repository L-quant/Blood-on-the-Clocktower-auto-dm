# agent

## 职责
AI 自动主持人 (Auto-DM) 系统：多代理编排、LLM 路由、记忆管理、工具调用，处理游戏事件并生成主持行为

## 成员文件
- `autodm.go` → Auto-DM 主入口，对外 API：事件处理、状态更新、启停控制 (convertEvent 优先读 nominator_user_id 修复代理提名)
- `autodm_test.go` → Auto-DM 创建、状态更新、事件处理、convertEvent nominator/PlayerID 修复测试
- `bridge.go` → 房间管理器桥接层，将 agent 工具操作转发到 RoomManager
- `tools.go` → 游戏工具定义与执行 (发消息、推进阶段等)
- `types.go` → 核心类型定义：Phase、Action、GameEvent、PlayerState、SubAgent 接口等
- `core/orchestrator.go` → 核心编排器，协调 5 个子代理处理事件
- `core/prompts.go` → 不同游戏阶段的系统提示词模板
- `llm/client.go` → OpenAI 兼容 LLM 客户端，自动检测 Gemini
- `llm/gemini.go` → Google Gemini API 客户端，含安全设置与重试
- `llm/router.go` → 按任务类型路由到不同 LLM 模型
- `memory/manager.go` → 短期记忆管理，事件追踪
- `subagent/moderator.go` → 主持子代理，管理游戏流程与提名验证
- `subagent/narrator.go` → 叙事子代理，生成氛围化游戏描述
- `subagent/player_modeler.go` → 玩家建模子代理，分析投票与指控行为
- `subagent/rules.go` → 规则子代理，回答规则问题与角色查询
- `subagent/summarizer.go` → 摘要子代理，生成游戏状态摘要
- `subagent/types.go` → 子代理共享类型：GameStateView、PlayerView 及格式化工具
- `tools/game_ops.go` → 游戏操作工具注册 (发消息、杀人、推进阶段等)
- `tools/registry.go` → 工具注册表，管理 LLM 可调用工具的定义与执行

## 对外接口
- `NewAutoDM(cfg Config) *AutoDM` → 创建 Auto-DM 实例
- `(*AutoDM) Start()` → 启动编排器
- `(*AutoDM) Stop()` → 停止编排器
- `(*AutoDM) IsActive() bool` → 返回是否活跃
- `(*AutoDM) Enabled() bool` → 返回是否启用
- `(*AutoDM) SetEnabled(enabled bool)` → 设置启用状态
- `(*AutoDM) SetDispatcher(dispatcher CommandDispatcher, stateGetter func() interface{})` → 配置命令分发器
- `(*AutoDM) SetCommander(commander tools.GameCommander)` → 设置游戏命令执行器
- `(*AutoDM) SetRulesProvider(rules tools.RulesProvider)` → 设置规则提供器
- `(*AutoDM) ProcessEvent(ctx context.Context, event Event) (*Response, error)` → 处理游戏事件
- `(*AutoDM) UpdateGameState(state *GameState)` → 更新游戏状态视图
- `(*AutoDM) GetSummary(ctx context.Context, forDM bool) (string, error)` → 获取游戏摘要
- `(*AutoDM) AnalyzePlayers(ctx context.Context) (string, error)` → 分析玩家行为
- `(*AutoDM) OnEvent(ctx context.Context, ev types.Event, state interface{})` → RoomActor 事件回调
- `(*AutoDM) ProcessQueuedEvent(ctx context.Context, ev types.Event) error` → 处理队列中的事件

## 依赖
- `internal/agent/core` → 核心编排器
- `internal/agent/llm` → LLM 客户端与路由
- `internal/agent/memory` → 短期记忆管理
- `internal/agent/subagent` → 五个子代理实现
- `internal/agent/tools` → 工具注册与执行
- `internal/engine` → 游戏状态类型 (State)
- `internal/game` → 角色定义与游戏上下文
- `internal/mcp` → MCP 工具注册表
- `internal/types` → 命令/事件信封类型
