# store

## 职责
Vuex 状态管理中心：8 个业务模块 + 2 个插件，管理游戏全部前端状态与后端通信

## 成员文件
- `index.js` → Vuex Store 主入口，组合所有模块和插件，含根级 state/mutations/actions/getters
- `helpers.js` → 工具函数：cleanId、isMobile、randomId、seatLabel
- `modules/game.js` → 游戏阶段、天数、胜者状态
- `modules/players.js` → 玩家列表、座位、角色、鬼牌、传奇角色
- `modules/annotations.js` → 玩家个人猜测标注 (按房间 localStorage 持久化)
- `modules/chat.js` → 多频道聊天 (公共/邪恶/密语/AI 助手)，未读计数
- `modules/night.js` → 夜晚行动覆盖层状态 (轮次、目标选择、进度)
- `modules/timeline.js` → 游戏事件时间线 (阶段变化、死亡、投票)
- `modules/vote.js` → 提名与投票状态 (提名者/被提名者/票数/结果/历史)
- `modules/ui.js` → UI 状态 (屏幕路由、标签页、弹窗、设置)
- `plugins/persistence.js` → localStorage 持久化插件 (设置/笔记/标注)
- `plugins/websocket.js` → WebSocket 插件：连接管理、事件→mutation 映射、命令发送、重连

## 对外接口
- `default` → Vuex Store 实例 (包含所有模块、插件和根级方法)
- `cleanId(id) → string` → 转为小写字母数字
- `isMobile() → boolean` → 检测视口 ≤ 768px
- `randomId() → string` → 生成 10 字符随机 ID
- `seatLabel(seatIndex) → string` → 格式化座位号为 "N号"

## 依赖
- `services/ApiService` → REST API 调用 (认证、建房、加入、状态同步)
- `i18n/` → 国际化实例 (通过 Vue 注入)
