# 夜晚逻辑三层架构重构

> 基于 `暗流涌动_夜晚逻辑需求文档.md` 的三层架构（收集-结算-分发），将原有"即时结算"模式改为延迟统一结算。

## Checklist

- [x] Phase 1: state.go 扩展 — Player 添加 SpyApparentRole，State 添加 ScarletWomanTriggered，更新 Copy()
- [x] Phase 2: game/night.go 间谍系统 — getApparentAlignment/getApparentRole + 修复 registersAsEvil 对 spy 的处理 + GrimoireSnapshot 结构
- [x] Phase 3: game/setup.go SpyApparentRole 赋值 — Setup 阶段从不在场善良角色中随机指定
- [x] Phase 4: engine.go handleAbility 重构 — 移除 ResolveAbility 调用，仅记录意图（targets），night.action.completed 不再携带 result
- [x] Phase 5: engine_night_resolve.go（新建）— resolveNight 统一结算：投毒→保护→击杀→死亡检查→红唇继承→投毒者死亡回滚
- [x] Phase 6: engine_night_info.go（新建）— distributeNightInfo 信息分发：team.recognition + spy grimoire + 所有信息角色 night.info
- [x] Phase 7: 集成调用链 — handleAbility 全部完成后调用 resolveNight + distributeNightInfo + phase.day；state_reduce 添加 night.info/team.recognition/poison.rollback reducer
- [x] Phase 8: projection.go — night.info 和 team.recognition 可见性规则 + sanitizePayload（strip is_false / spy_apparent_role / bluffs for minions）
- [x] Phase 9: 前端 ws_game_events.js — 处理 night.info 和 team.recognition 事件
- [x] Phase 10: 前端 night.js + NightOverlay — night.action.completed 不再显示 result；night.info 驱动 result 显示
- [x] Phase 11: 编译验证与测试 — go build 通过、前端无 lint 错误
- [x] 回环检查：更新所有受影响的 CLAUDE.md 和文件头注释

## 状态：✅ 全部完成
