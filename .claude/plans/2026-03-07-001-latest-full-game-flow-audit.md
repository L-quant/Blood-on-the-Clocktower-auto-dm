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
  - 每一步留存截图、状态与事件证据

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
- 证据：
  - G1 截图：`.playwright-mcp/2026-03-07-manual-game1/12-night-progress-20s.png`
  - G2 截图：`.playwright-mcp/2026-03-07-manual-game2/12-night-20s.png`
  - G3 截图：`.playwright-mcp/2026-03-07-manual-game3/11-night-60s-total.png`
  - 对应状态：
    - `.playwright-mcp/2026-03-07-manual-game1/state-night-stuck.json`
    - `.playwright-mcp/2026-03-07-manual-game2/state-night-stuck.json`
    - `.playwright-mcp/2026-03-07-manual-game3/state-night-stuck.json`
- 后端状态共性：`night_actions` 中存在 `completed=false` 的 bot 行动，同时 `timed_out=null`，房间持续 `phase=night`。
- 影响：阻断级，当前版本无法完成“完整一局”。
- 建议：
  1. 修复 bot 夜晚行动提交链路（至少保证 info/select 类行动可完成）。
  2. 为 night 增加兜底超时/强制收敛机制（即使 `NIGHT_ACTION_TIMEOUT_SEC=0` 也应避免死锁）。
  3. 在 UI 增加“夜晚等待来源/剩余玩家”提示，避免玩家误判为前端卡死。

### P0-02 默认前后端端口配置不一致，默认启动不可玩
- 现象：前端默认请求 `localhost:8888`，后端默认监听 `:8080`，创建房间直接失败。
- 证据：
  - `.playwright-mcp/2026-03-07-default-config-check/shot-0.png`
  - `.playwright-mcp/2026-03-07-default-config-check/errors-0.json`（`ERR_CONNECTION_REFUSED` / `ERR_CONNECTION_CLOSED`）
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

## 本轮证据目录
- `.playwright-mcp/2026-03-07-manual-game1/`
- `.playwright-mcp/2026-03-07-manual-game2/`
- `.playwright-mcp/2026-03-07-manual-game3/`
- `.playwright-mcp/2026-03-07-default-config-check/`

## 修复优先级建议
1. 先修 P0-01 夜晚死锁（不修复无法完成整局）
2. 再修 P0-02 默认端口配置（不修复默认不可玩）
3. 修 P1-02 图标缺失（降低噪音、恢复状态反馈）
4. 优化 P1-01 开局阶段反馈与耗时体验

## 状态
- ✅ 已完成：3 局真实浏览器完整链路复测（到夜晚阻断点）
- ✅ 已完成：问题归档与证据落盘
- ⏭ 待执行：按优先级进入修复与回归
