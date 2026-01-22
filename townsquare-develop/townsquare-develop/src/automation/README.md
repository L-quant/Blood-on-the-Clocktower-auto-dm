# 血染钟楼自动化说书人系统

## 概述

自动化说书人系统是一个智能化的游戏管理解决方案，能够自动处理游戏流程、角色能力、投票管理和AI决策支持。

## 架构

系统采用分层架构：

```
用户界面层 (Vue Components)
    ↓
业务逻辑层 (Vuex Store)
    ↓
游戏引擎层 (Core Modules)
    ↓
数据持久层 (Configuration & State)
```

## 核心模块

### 1. AutomatedStorytellerSystem
主控制器，协调所有子系统的工作。

### 2. GameStateManager
管理游戏状态和阶段转换。

### 3. RoleAssigner
自动分配角色给玩家。

### 4. NightActionProcessor
处理夜间阶段的所有角色能力。

### 5. VotingManager
管理白天讨论和投票流程。

### 6. VictoryConditionChecker
检查和判断游戏胜负条件。

### 7. AIDecisionEngine
为恶人阵营提供智能决策支持。

### 8. AbilityResolver & AbilityExecutor
解析和执行角色能力。

### 9. StateSynchronizer
确保所有客户端状态同步。

### 10. ConfigurationManager
管理游戏配置和日志。

### 11. RolePrivacySystem（无说书人模式）
管理角色隐私保护，确保每个玩家只能看到自己的角色。

### 12. PerspectiveFilter（无说书人模式）
根据玩家身份过滤游戏状态中的敏感信息。

### 13. PlayerAuthenticator（无说书人模式）
管理玩家身份验证和座位认领。

### 14. ModeSwitcher（无说书人模式）
处理游戏模式切换。

## Vue组件集成

### AutomationPanel组件

自动化控制面板，提供用户界面来管理自动化系统。

#### 使用方法

1. 在Menu中点击"自动化说书人"标签
2. 点击"打开控制面板"
3. 在面板中初始化系统
4. 配置AI难度和自动化级别
5. 启动自动化游戏

#### 功能特性

- **系统状态监控**: 实时显示系统运行状态
- **游戏阶段显示**: 显示当前游戏阶段（准备、第一夜、白天、夜晚）
- **控制按钮**: 启动、暂停、恢复、停止自动化
- **AI决策建议**: 显示待处理的AI决策建议
- **配置管理**: 调整AI难度、自动化级别、调试模式
- **错误显示**: 显示系统错误信息
- **日志查看**: 调试模式下查看系统日志

### Vuex Store集成

自动化系统通过Vuex store模块进行状态管理：

```javascript
// 访问状态
this.$store.state.automation.systemStatus
this.$store.state.automation.isAutomationEnabled

// 使用getters
this.$store.getters['automation/isSystemRunning']
this.$store.getters['automation/pendingDecisionCount']

// 调用actions
this.$store.dispatch('automation/initializeAutomation', config)
this.$store.dispatch('automation/startAutomation')
this.$store.dispatch('automation/stopAutomation')

// 提交mutations
this.$store.commit('automation/SET_SYSTEM_STATUS', status)
this.$store.commit('automation/ADD_LOG', log)
```

## 使用流程

### 1. 初始化系统

```javascript
// 在AutomationPanel中
await this.initializeAutomation()
```

这将：
- 创建AutomatedStorytellerSystem实例
- 初始化所有子系统
- 验证配置
- 更新Vuex状态

### 2. 启动自动化游戏

```javascript
// 确保至少有5名玩家
await this.startAutomation()
```

这将：
- 初始化游戏状态
- 自动分配角色
- 转换到第一夜
- 处理第一夜行动
- 开始游戏循环

### 3. 游戏循环

系统自动处理：
- 夜间行动（按官方顺序）
- 白天讨论和投票
- 胜负条件检查
- 阶段转换

### 4. AI决策支持

当恶人玩家需要做决策时：
- 系统分析当前游戏状态
- 生成多个决策建议
- 显示在控制面板中
- 玩家可以选择应用建议

### 5. 暂停和恢复

```javascript
// 暂停自动化
await this.pauseAutomation()

// 恢复自动化
await this.resumeAutomation()
```

### 6. 停止游戏

```javascript
await this.stopAutomation()
```

## 配置选项

### 游戏模式
- **说书人模式**: 传统模式，说书人可以看到所有角色
- **无说书人模式**: 玩家直接游玩，每个玩家只能看到自己的角色

### AI难度
- **简单**: 基础决策，适合新手
- **中等**: 平衡的决策，适合普通玩家
- **困难**: 高级决策，适合经验丰富的玩家

### 自动化级别
- **手动**: 需要手动确认每个操作
- **半自动**: 自动处理部分操作，关键决策需要确认
- **全自动**: 完全自动化，无需人工干预

### 调试模式
启用后显示详细的系统日志，用于开发和调试。

## 无说书人模式

### 概述

无说书人模式允许玩家直接游玩血染钟楼，无需额外的说书人。系统自动处理游戏流程，同时确保每个玩家只能看到自己的角色信息。

### 核心特性

1. **角色隐私保护**: 每个玩家只能看到自己的角色
2. **座位认领系统**: 玩家通过令牌认领座位
3. **身份验证**: 确保玩家身份的安全性
4. **断线重连**: 支持玩家断线后重新连接
5. **游戏结束公开**: 游戏结束时自动公开所有角色

### 使用流程

#### 1. 选择游戏模式

在游戏开始前，通过ModeSelector组件选择"无说书人模式"。

#### 2. 认领座位

每个玩家需要认领一个座位：
1. 点击空座位
2. 选择"认领座位"
3. 系统生成唯一令牌
4. 座位被绑定到玩家

#### 3. 游戏进行

- 系统自动分配角色
- 每个玩家只能看到自己的角色卡正面
- 其他玩家的角色卡显示为背面
- 夜间信息只发送给相关玩家
- 白天阶段公开死亡信息（但不透露角色）

#### 4. 游戏结束

游戏结束时：
- 自动解除隐私保护
- 所有角色卡翻转显示
- 显示完整的游戏报告

### 隐私保护机制

#### 视角过滤

PerspectiveFilter根据玩家身份过滤游戏状态：
- **公开信息**: 所有玩家可见（存活状态、投票结果）
- **私密信息**: 只有相关玩家可见（夜间信息、能力结果）
- **角色信息**: 只有自己可见（除非游戏结束）

#### 访问控制

RolePrivacySystem记录所有访问尝试：
- 合法访问：玩家访问自己的角色
- 非法访问：玩家尝试访问其他角色（被阻止并记录）

#### 安全日志

系统记录所有安全相关事件：
- 座位认领
- 令牌验证
- 访问尝试
- 安全违规

### 断线重连

玩家断线后可以重新连接：
1. 使用原有的令牌
2. 系统验证令牌有效性
3. 恢复座位绑定
4. 同步游戏状态

### 配置示例

```javascript
const config = {
  scriptType: 'trouble-brewing',
  playerCount: 7,
  gameMode: 'player-only',  // 启用无说书人模式
  automationLevel: 'full_auto',
  aiDifficulty: 'medium'
};

await system.initialize(config);
```

### Vuex Store集成

无说书人模式通过privacy模块管理状态：

```javascript
// 访问状态
this.$store.state.privacy.gameMode
this.$store.state.privacy.myPlayerId
this.$store.state.privacy.mySeatIndex
this.$store.state.privacy.myRole

// 使用getters
this.$store.getters['privacy/isPlayerOnlyMode']
this.$store.getters['privacy/canSeeRole'](playerId)
this.$store.getters['privacy/hasClaimedSeat']

// 调用actions
this.$store.dispatch('privacy/claimSeat', { playerId, seatIndex })
this.$store.dispatch('privacy/releaseSeat')
this.$store.dispatch('privacy/switchGameMode', 'player-only')
```

### 私密信息显示

PrivateInfo组件显示玩家的私密信息：
- 我的角色
- 夜间信息
- 能力结果

该组件只在无说书人模式下显示，并且只显示当前玩家的信息。

### 安全考虑

1. **令牌安全**: 使用唯一令牌标识玩家身份
2. **访问控制**: 严格限制角色信息访问
3. **日志记录**: 记录所有安全相关事件
4. **加密存储**: 敏感信息加密存储（可选）
5. **超时机制**: 长时间断线自动释放座位

### 限制和注意事项

1. **游戏开始后无法切换模式**: 确保在游戏开始前选择正确的模式
2. **座位唯一性**: 每个座位只能被一个玩家认领
3. **令牌有效期**: 令牌有时间限制，过期需要重新认领
4. **网络要求**: 需要稳定的网络连接以保持状态同步

## 事件系统

### 全局事件

```javascript
// 打开自动化面板
this.$root.$emit('open-automation-panel')
```

### Vuex事件

系统通过Vuex mutations和actions触发状态变化，所有组件都可以响应这些变化。

## 错误处理

系统提供完善的错误处理机制：

1. **错误捕获**: 所有操作都包含try-catch
2. **错误记录**: 错误自动记录到Vuex store
3. **错误显示**: 在控制面板中显示错误信息
4. **错误恢复**: 提供清除错误和重试机制

## 测试

### 运行测试

```bash
# 运行所有测试
npm test

# 运行组件测试
npm test -- tests/automation/components

# 运行核心模块测试
npm test -- tests/automation/core
```

### 测试覆盖

- 核心模块: 421个测试
- Vue组件: 18个测试
- 总计: 439个测试

## 开发指南

### 添加新功能

1. 在核心模块中实现功能
2. 在Vuex store中添加状态和actions
3. 在Vue组件中添加UI
4. 编写测试
5. 更新文档

### 调试技巧

1. 启用调试模式查看详细日志
2. 使用Vue DevTools查看Vuex状态
3. 在浏览器控制台查看错误信息
4. 使用断点调试核心模块

## API参考

### AutomatedStorytellerSystem

```javascript
// 初始化
await system.initialize(config)

// 启动游戏
await system.startAutomatedGame(players)

// 控制
system.pauseAutomation()
system.resumeAutomation()
system.stopGame()

// 获取状态
const status = system.getSystemStatus()

// AI决策
const suggestions = system.getAIDecisionSuggestions(context)
const report = system.getAIDecisionReport(context)

// 紧急处理
system.handleEmergency(emergencyType)
```

### Vuex Store Actions

```javascript
// 初始化
dispatch('automation/initializeAutomation', config)

// 控制
dispatch('automation/startAutomation')
dispatch('automation/stopAutomation')
dispatch('automation/pauseAutomation')
dispatch('automation/resumeAutomation')

// 错误处理
dispatch('automation/handleError', error)
```

### Vuex Store Mutations

```javascript
// 状态管理
commit('automation/SET_SYSTEM_STATUS', status)
commit('automation/SET_AUTOMATION_ENABLED', enabled)
commit('automation/SET_CURRENT_PHASE', phase)

// 决策管理
commit('automation/ADD_AI_DECISION', decision)
commit('automation/ADD_PENDING_DECISION', decision)
commit('automation/REMOVE_PENDING_DECISION', decisionId)

// 日志和错误
commit('automation/ADD_LOG', log)
commit('automation/ADD_ERROR', error)
commit('automation/CLEAR_ERRORS')

// 配置
commit('automation/UPDATE_CONFIGURATION', config)
```

## 性能优化

1. **状态更新**: 使用Vuex确保高效的状态管理
2. **日志限制**: 自动限制日志数量，避免内存泄漏
3. **异步处理**: 所有耗时操作都是异步的
4. **错误恢复**: 快速从错误中恢复，不影响游戏流程

## 安全考虑

1. **输入验证**: 所有用户输入都经过验证
2. **状态保护**: 关键状态变化需要权限检查
3. **错误隔离**: 错误不会影响其他模块
4. **日志脱敏**: 敏感信息不会记录到日志

## 未来计划

- [ ] 支持更多官方脚本
- [ ] 增强AI决策算法
- [ ] 添加游戏回放功能
- [ ] 支持自定义规则
- [ ] 多语言支持
- [ ] 移动端优化

## 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork项目
2. 创建功能分支
3. 编写代码和测试
4. 提交Pull Request
5. 等待代码审查

## 许可证

本项目遵循原项目的许可证。

## 联系方式

如有问题或建议，请通过以下方式联系：

- GitHub Issues
- Discord社区
- 项目邮件列表
