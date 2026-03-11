# 2026-03-07 最新代码完整流程审计（基于 origin/main@cbfb9fc）

## 目标
- 基于远端最新代码，从真实玩家视角验证“完整一局”是否可走通。
- 多局复测并记录所有阻断问题与体验问题。

## 本轮环境与方法（真实浏览器操作）
- 代码基线：`main` = `origin/main` = `cbfb9fc`
- 后端：`http://localhost:8080`
- 前端：`http://localhost:8092`（为避免默认端口不一致导致无法联调，临时使用 8080 API/WS）
- 测试方式：
  - 使用安装的 `develop-web-game` skill 对应 Playwright 环境，开启可见浏览器（headful）
  - 在同一浏览器会话中逐步点击完成玩家路径（非 loadtest）
  - 每一步记录状态与事件证据，并将关键证据内嵌到本文件

## 覆盖路径
- 首页创建房间
- 大厅入座/补足 7 人（1 真人 + 6 bot）
- 房主开始游戏
- 首夜/白天（视角色和时序而定）
- 提名 + 辩护 + 顺序投票
- 推进到下一夜并观察是否可继续

## 多局结果摘要

| 局次 | 房间ID | 关键流程 | 结果 |
|---|---|---|---|
| G1 | `d8e44b08-77ab-4ff1-a87f-4b9d47b39bd5` | 开局→白天→提名投票→进夜 | 夜晚卡死（>20s） |
| G2 | `00b64b2b-9092-470b-acb3-477b659d2721` | 开局→首夜交互→白天投票→进夜 | 夜晚卡死（>20s） |
| G3 | `91830dd2-e320-4c03-8de5-68102700d000` | 开局→白天投票→进夜 | 夜晚卡死（>60s） |

结论：**3/3 局在“第一天后进入 night”阶段无法继续推进，完整一局不可完成。**

## 问题清单（按优先级）

### P0-01 进入夜晚后流程卡死，整局无法完成（100% 复现）
- 现象（玩家视角）：
  - 提名投票后进入夜晚，界面长期停留在“夜幕降临…闭上眼睛，等待被唤醒”。
  - 无可用前端操作可继续推进，玩家体验为“整局被硬锁死”。
- 本轮复现：G1/G2/G3 全部复现。
- 证据（状态快照摘要）：
  - G1：`phase=night`, `day_count=1`, `night_count=2`, 未完成夜间行动 1 个（`info`，`timed_out=null`）
  - G2：`phase=night`, `day_count=1`, `night_count=2`, 未完成夜间行动 2 个（均为 `info`，`timed_out=null`）
  - G3：`phase=night`, `day_count=1`, `night_count=2`, 未完成夜间行动 2 个（`select_two` + `info`，`timed_out=null`）
- 证据（事件链摘要）：
  - 3 局都出现：`phase.day -> nomination.created -> nomination.resolved -> phase.night -> night.action.prompt(...)`，随后停滞在 `phase.night`
<<<<<<< HEAD
- 后端状态共性：`night_actions` 中存在 `completed=false` 的 bot 行动，同时 `timed_out=null`，房间持续 `phase=night`。
- 影响：阻断级，当前版本无法完成“完整一局”。
- 建议：
  1. 修复 bot 夜晚行动提交链路（至少保证 info/select 类行动可完成）。
  2. 为 night 增加兜底超时/强制收敛机制（即使 `NIGHT_ACTION_TIMEOUT_SEC=0` 也应避免死锁）。
  3. 在 UI 增加“夜晚等待来源/剩余玩家”提示，避免玩家误判为前端卡死。
=======
  - 后端状态共性：`night_actions` 中存在 `completed=false` 的 bot 行动，同时 `timed_out=null`，房间持续 `phase=night`。
- 影响：阻断级，当前版本无法完成“完整一局”。
>>>>>>> feature/major-gameplay-fixes

### P0-02 默认前后端端口配置不一致，默认启动不可玩
- 现象：前端默认请求 `localhost:8888`，后端默认监听 `:8080`，创建房间直接失败。
- 证据（控制台错误）：
  - `Failed to load resource: net::ERR_CONNECTION_REFUSED`
  - `Failed to load resource: net::ERR_CONNECTION_CLOSED`
- 影响：新环境默认体验即失败。
- 建议：统一默认 API/WS 地址（或增加 dev proxy）。

### P1-01 开始游戏耗时长且反馈弱（约 20~30 秒）
- 现象：点击“开始游戏”后较长时间仍停留大厅，玩家难以判断是否成功触发。
- 证据：
  - G1: `03-after-start.png` + `04-after-start-wait20s.png`
  - G2/G3: `03-after-start-24s.png`
- 影响：高感知延迟，易引发重复点击和误操作。
- 建议：增加明确阶段化进度（例如“正在分配角色/正在同步玩家/即将进入首夜”）。

### P1-02 运行期图标缺失（spinner/moon）
- 现象：控制台持续报错未注册图标。
- 本轮捕获：
  - `Could not find ... iconName: spinner`
  - `Could not find ... iconName: moon`
- 影响：视觉反馈降级，且污染控制台，影响调试。
- 建议：在 `frontend/src/main.js` 补齐对应 FontAwesome 图标注册。

<<<<<<< HEAD
=======
## 与当前分支实现差异（截至 `a11e2e1`）

本节用于说明：本审计文档记录的是 `origin/main@cbfb9fc` 的人工联调结果，不等于当前工作分支的最终产品决策。当前分支已经对部分问题做了策略调整，但并未把本审计中的所有建议原样采纳。

### 逐项状态

#### P0-01 进入夜晚后流程卡死，整局无法完成
- 现状：**根因仍未确认已彻底修复**。
- 当前分支已做的相关变更：
  - 后端明确禁止在夜晚通过 `advance_phase("day")` 强制切到白天。
  - 后端明确禁用 `night_timeout` 路径，不再允许用夜晚超时来“兜底收夜”。
  - 房间层不再调度或恢复夜晚超时计时器。
- 结论：
  - 本问题在当前分支里**不是通过“让夜晚更容易结束”来修**，而是改成“夜晚只能自然结束”。
  - 因此，本审计中针对该问题的建议 `2. 为 night 增加兜底超时/强制收敛机制` 已被**明确否决**。
  - 建议 `1. 修复 bot 夜晚行动提交链路` 与 `3. 增加夜晚等待来源/剩余玩家提示` 仍属于**待处理项**。

#### P0-02 默认前后端端口配置不一致，默认启动不可玩
- 现状：**未修**。
- 依据：
  - `frontend/.env.local` 仍指向 `localhost:8888`。
  - `frontend/src/services/ApiService.js` 默认 REST 地址仍回退到 `http://localhost:8888`。
  - `frontend/src/store/plugins/websocket.js` 默认 WS 地址回退到 `ws://localhost:8080/ws`。
- 结论：当前分支仍存在默认 REST/WS 端口不一致的问题，本审计反馈依然成立。

#### P1-01 开始游戏耗时长且反馈弱
- 现状：**未修**。
- 依据：当前分支未见针对“开始游戏阶段反馈”新增明确的进度态、分步提示或加载文案。
- 结论：本审计反馈仍成立。

#### P1-02 运行期图标缺失（spinner/moon）
- 现状：**未修**。
- 依据：`frontend/src/main.js` 当前仍未注册 `spinner` / `moon` 对应图标。
- 结论：本审计反馈仍成立。

### 当前分支额外已完成项（不在本审计原始问题单内）

以下变更已经在当前分支落地，但并非本审计文档原始列出的反馈项：

- 为间谍新增“魔典”历史展示：在事件/记录侧栏按夜 Accordion 展示完整魔典快照。
- 夜晚查验记录链路增强：普通 `night.info` 与间谍 `grimoire` 分开存储和展示。
- 投票结算播报增强：公开聊天中可显示“X号提名Y号，投票玩家……”摘要。
- 聊天界面文案和样式修正：`公开/悄悄话` 已调整为 `公聊/私聊`，私聊对象下拉背景已修复。

### 本文档解读方式

- 这份文档仍然保留为“上游基线版本的人工测试证据”。
- 若用于指导当前分支开发，请优先参考本节状态，而不是把上文建议视为当前分支的既定方案。
- 尤其是 P0-01 的“夜晚超时/强制收敛”建议，已与当前分支策略冲突，不应继续按该建议实施。

>>>>>>> feature/major-gameplay-fixes
## 本轮证据目录
- 原始截图/JSON 已用于归纳并内嵌关键结论；如需追查，可按房间 ID 在后端事件库复盘。

## 关键原始证据摘录（内嵌）
### G1 夜晚卡死状态
```json
{
  "room_id": "d8e44b08-77ab-4ff1-a87f-4b9d47b39bd5",
  "phase": "night",
  "day_count": 1,
  "night_count": 2,
  "pending_night_actions": [
    {
      "user_id": "bot-9cd7b5c0",
      "action_type": "info",
      "completed": false,
      "timed_out": null
    }
  ]
}
```

### G2 夜晚卡死状态
```json
{
  "room_id": "00b64b2b-9092-470b-acb3-477b659d2721",
  "phase": "night",
  "day_count": 1,
  "night_count": 2,
  "pending_night_actions": [
    {
      "user_id": "bot-039b5a1b",
      "action_type": "info",
      "completed": false,
      "timed_out": null
    },
    {
      "user_id": "bot-58577191",
      "action_type": "info",
      "completed": false,
      "timed_out": null
    }
  ]
}
```

### G3 夜晚卡死状态
```json
{
  "room_id": "91830dd2-e320-4c03-8de5-68102700d000",
  "phase": "night",
  "day_count": 1,
  "night_count": 2,
  "pending_night_actions": [
    {
      "user_id": "bot-9ce8d5c1",
      "action_type": "select_two",
      "completed": false,
      "timed_out": null
    },
    {
      "user_id": "bot-5f379b09",
      "action_type": "info",
      "completed": false,
      "timed_out": null
    }
  ]
}
```

### 端口不一致时的错误摘录
```json
[
  {
    "type": "console.error",
    "text": "Failed to load resource: net::ERR_CONNECTION_REFUSED"
  },
  {
    "type": "console.error",
    "text": "Failed to load resource: net::ERR_CONNECTION_CLOSED"
  }
]
```

## 修复优先级建议
1. 先修 P0-01 夜晚死锁（不修复无法完成整局）
2. 再修 P0-02 默认端口配置（不修复默认不可玩）
3. 修 P1-02 图标缺失（降低噪音、恢复状态反馈）
4. 优化 P1-01 开局阶段反馈与耗时体验

## 状态
- ✅ 已完成：3 局真实浏览器完整链路复测（到夜晚阻断点）
- ✅ 已完成：问题归档与证据落盘
- ⏭ 待执行：按优先级进入修复与回归
