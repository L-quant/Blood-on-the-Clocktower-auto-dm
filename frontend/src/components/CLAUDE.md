# components

## 职责
24 个 Vue 单文件组件，构成游戏全部 UI 界面：首页、大厅、游戏主屏、夜晚/投票覆盖层、结算

## 成员文件
- `HomeScreen.vue` → 首页：创建/加入房间入口
- `JoinRoomSheet.vue` → 底部弹出的房间号输入面板
- `LobbyScreen.vue` → 大厅：房间码、座位格、游戏配置 (房主)
- `LobbyPlayerGrid.vue` → 大厅座位格，显示占用状态
- `LobbyPlayerSlot.vue` → 单个座位槽，含占用标识与房主徽章
- `EditionPicker.vue` → 剧本选择器
- `GameScreen.vue` → 游戏主布局，移动端 Tab 切换 / 桌面端多栏
- `SquareView.vue` → 城镇广场容器，包含玩家圆圈与行动面板
- `PlayerCircle.vue` → 圆形排列的玩家节点布局
- `PlayerNode.vue` → 单个玩家令牌：角色图标、死亡阴影、提名脉冲
- `PlayerActionSheet.vue` → 玩家操作底部面板 (标注角色、笔记、提名)
- `RoleAnnotator.vue` → 角色猜测选择器 (阵营 Tab + 角色网格)
- `AliveCounter.vue` → 头部存活/死亡计数条
- `ChatView.vue` → 多频道聊天 (公共/密语/邪恶/AI 助手)
- `TimelineView.vue` → 事件时间线 (含过滤按钮)
- `MeView.vue` → 个人角色展示：能力、鬼牌、笔记
- `NightOverlay.vue` → 夜晚行动界面：按 actionType 分流 (select→选人, info/passive→自动提交, no_action→确认跳过)
- `VoteOverlay.vue` → 投票界面：提名信息、进度条、投票按钮
- `PhaseTransition.vue` → 全屏阶段切换动画通知
- `TopBar.vue` → 顶部栏：连接状态、阶段信息、房间码、设置
- `BottomNav.vue` → 底部导航栏 (含未读消息徽章)
- `SettingsPanel.vue` → 设置面板 (音效、动画、语言)
- `ConfirmDialog.vue` → 确认对话框
- `GameEndScreen.vue` → 结算屏幕 (胜方展示与返回)

## 对外接口
- 所有组件通过 Vue 组件注册导出，无独立函数 API
- 组件间通过 props/events 或 Vuex store 通信

## 依赖
- `store/modules/game` → 游戏阶段、胜负状态
- `store/modules/players` → 玩家列表、角色、存活状态
- `store/modules/chat` → 聊天消息、频道切换
- `store/modules/vote` → 提名与投票状态
- `store/modules/night` → 夜晚行动状态
- `store/modules/timeline` → 事件时间线
- `store/modules/ui` → 屏幕路由、弹窗、设置
- `store/modules/annotations` → 玩家角色猜测标注
- `services/SoundService` → 音效控制 (SettingsPanel)
