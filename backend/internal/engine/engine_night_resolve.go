// engine_night_resolve.go — 夜晚统一结算层
//
// 所有夜晚行动收集完毕后，按官方结算顺序统一处理：
// 投毒者→僧侣→小恶魔→红唇女郎继承→投毒者死亡回滚
//
// [IN]  internal/game（角色定义）
// [IN]  internal/types（Event 类型）
// [POS] 三层架构的结算层，与 engine_night_info.go（分发层）配合
package engine

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// resolveNight 统一结算所有已收集的夜晚行动意图。
// 返回效果事件列表（player.died / player.poisoned / player.protected / demon.changed）。
func resolveNight(state State, cmd types.CommandEnvelope) []types.Event {
	events := []types.Event{}
	isFirstNight := state.Phase == PhaseFirstNight

	// 构建行动意图映射：role -> NightAction（含 targets）
	intentByRole := buildIntentMap(state)

	// === 第一步：投毒者结算 ===
	poisonTargetID := ""
	poisonerID := ""
	if intent, ok := intentByRole["poisoner"]; ok && len(intent.TargetIDs) > 0 {
		poisonTargetID = intent.TargetIDs[0]
		poisonerID = intent.UserID
		events = append(events, newEvent(cmd, "player.poisoned", map[string]string{
			"user_id": poisonTargetID,
		}))
		slog.Info("night.resolve: poisoner applied",
			"target", poisonTargetID, "poisoner", poisonerID)
	}

	// === 第二步：僧侣保护结算（非首夜）===
	protectTargetID := ""
	monkID := ""
	if !isFirstNight {
		if intent, ok := intentByRole["monk"]; ok && len(intent.TargetIDs) > 0 {
			protectTargetID = intent.TargetIDs[0]
			monkID = intent.UserID
			events = append(events, newEvent(cmd, "player.protected", map[string]string{
				"user_id": protectTargetID,
			}))
			slog.Info("night.resolve: monk protected",
				"target", protectTargetID, "monk", monkID)
		}
	}

	// === 第三步：管家选主人 ===
	if intent, ok := intentByRole["butler"]; ok && len(intent.TargetIDs) > 0 {
		events = append(events, newEvent(cmd, "reminder.added", map[string]string{
			"user_id":  intent.UserID,
			"reminder": fmt.Sprintf("master:%s", intent.TargetIDs[0]),
		}))
	}

	// 首夜不执行击杀，直接返回
	if isFirstNight {
		return events
	}

	// === 第四步：小恶魔击杀结算 ===
	if intent, ok := intentByRole["imp"]; ok && len(intent.TargetIDs) > 0 {
		killTargetID := intent.TargetIDs[0]
		demonID := intent.UserID

		// 恶魔被毒 → 刀无效（官方规则：中毒的恶魔无法杀人）
		demonPoisoned := demonID == poisonTargetID
		if demon, ok := state.Players[demonID]; ok && (demonPoisoned || demon.IsPoisoned) {
			slog.Info("night.resolve: imp poisoned, kill negated",
				"demon", demonID, "target", killTargetID)
		} else {
			slog.Info("night.resolve: imp attacks", "target", killTargetID, "demon", demonID)
			killEvents := resolveDemonKill(killTargetID, demonID, poisonTargetID,
				protectTargetID, monkID, state, cmd)
			events = append(events, killEvents...)
		}
	}

	// === 第五步：投毒者死亡回滚 ===
	// 如果投毒者今晚被杀，其投毒效果应被回滚
	if poisonerID != "" {
		poisonerDied := isPlayerDiedInEvents(poisonerID, events)
		if poisonerDied && poisonTargetID != "" {
			events = append(events, newEvent(cmd, "poison.rollback", map[string]string{
				"user_id": poisonTargetID,
				"reason":  "poisoner_died",
			}))
			slog.Info("night.resolve: poison rollback",
				"poisoner", poisonerID, "target", poisonTargetID)
		}
	}

	return events
}

// resolveDemonKill 处理恶魔击杀的完整优先级链。
func resolveDemonKill(targetID, demonID, poisonTargetID, protectTargetID, monkID string,
	state State, cmd types.CommandEnvelope) []types.Event {

	events := []types.Event{}
	target, exists := state.Players[targetID]
	if !exists {
		return events
	}

	// 自杀：触发红唇女郎继承检查
	if targetID == demonID {
		events = append(events, resolveStarpass(demonID, state, cmd)...)
		return events
	}

	// 优先级 1：士兵免疫（中毒时失效）
	targetPoisoned := targetID == poisonTargetID
	if target.TrueRole == "soldier" && !targetPoisoned && !target.IsPoisoned {
		slog.Info("night.resolve: soldier immune", "target", targetID)
		return events
	}

	// 优先级 2：僧侣保护（僧侣自身中毒时保护无效）
	monkPoisoned := monkID == poisonTargetID
	if targetID == protectTargetID && monkID != "" {
		if monk, ok := state.Players[monkID]; ok && !monkPoisoned && !monk.IsPoisoned {
			slog.Info("night.resolve: monk protection effective", "target", targetID)
			return events
		}
	}

	// 优先级 3：镇长转移
	if target.TrueRole == "mayor" && !targetPoisoned && !target.IsPoisoned {
		bounceTarget := selectMayorBounceTarget(targetID, demonID, state)
		if bounceTarget != "" {
			slog.Info("night.resolve: mayor bounce", "from", targetID, "to", bounceTarget)
			events = append(events, newEvent(cmd, "player.died", map[string]string{
				"user_id": bounceTarget,
				"cause":   "demon_mayor_bounce",
			}))
			return events
		}
	}

	// 默认：目标死亡
	events = append(events, newEvent(cmd, "player.died", map[string]string{
		"user_id": targetID,
		"cause":   "demon",
	}))

	return events
}

// resolveStarpass 处理小恶魔自杀触发的红唇女郎继承。
func resolveStarpass(demonID string, state State, cmd types.CommandEnvelope) []types.Event {
	events := []types.Event{}

	// 老恶魔死亡
	events = append(events, newEvent(cmd, "player.died", map[string]string{
		"user_id": demonID,
		"cause":   "starpass",
	}))

	// 检查红唇女郎继承条件
	if state.ScarletWomanTriggered {
		return events
	}

	aliveCount := state.GetAliveCount()
	// 自杀后存活数要减 1
	aliveCount--
	if aliveCount < 5 {
		slog.Info("night.resolve: starpass but alive<5, no inherit")
		return events
	}

	// 优先找红唇女郎
	var scarletWomanID string
	var candidateMinions []string
	for _, minionID := range state.MinionIDs {
		p := state.Players[minionID]
		if p.Alive && minionID != demonID {
			candidateMinions = append(candidateMinions, minionID)
			if p.TrueRole == "scarletwoman" {
				scarletWomanID = minionID
			}
		}
	}

	if len(candidateMinions) == 0 {
		return events
	}

	newDemonID := scarletWomanID
	if newDemonID == "" {
		// 没有红唇女郎，随机选一个存活爪牙
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(candidateMinions))))
		if err != nil {
			newDemonID = candidateMinions[0]
		} else {
			newDemonID = candidateMinions[idx.Int64()]
		}
	}

	events = append(events, newEvent(cmd, "demon.changed", map[string]string{
		"old_demon": demonID,
		"new_demon": newDemonID,
		"reason":    "scarletwoman",
	}))

	slog.Info("night.resolve: starpass inherit",
		"old_demon", demonID, "new_demon", newDemonID)
	return events
}

// selectMayorBounceTarget 为镇长转移选择一个随机存活非恶魔玩家。
func selectMayorBounceTarget(mayorID, demonID string, state State) string {
	var candidates []string
	for uid, p := range state.Players {
		if uid != mayorID && uid != demonID && p.Alive && !p.IsDM {
			candidates = append(candidates, uid)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(candidates))))
	return candidates[idx.Int64()]
}

// buildIntentMap 从 NightActions 构建 role -> action 的意图映射。
func buildIntentMap(state State) map[string]NightAction {
	m := make(map[string]NightAction)
	for _, a := range state.NightActions {
		if a.Completed {
			m[a.RoleID] = a
		}
	}
	return m
}

// isPlayerDiedInEvents 检查给定事件列表中是否包含指定玩家的死亡事件。
func isPlayerDiedInEvents(userID string, events []types.Event) bool {
	for _, e := range events {
		if e.EventType == "player.died" {
			var payload map[string]string
			_ = json.Unmarshal(e.Payload, &payload)
			if payload["user_id"] == userID {
				return true
			}
		}
	}
	return false
}

// applyResolveEffects 将结算层产生的事件效果应用到 state 副本上，
// 供信息分发层使用最终状态。
func applyResolveEffects(state *State, events []types.Event) {
	for _, e := range events {
		var payload map[string]string
		_ = json.Unmarshal(e.Payload, &payload)

		switch e.EventType {
		case "player.died":
			uid := payload["user_id"]
			if p, ok := state.Players[uid]; ok {
				p.Alive = false
				state.Players[uid] = p
			}
		case "player.poisoned":
			uid := payload["user_id"]
			if p, ok := state.Players[uid]; ok {
				p.IsPoisoned = true
				state.Players[uid] = p
			}
		case "poison.rollback":
			uid := payload["user_id"]
			if p, ok := state.Players[uid]; ok {
				p.IsPoisoned = false
				state.Players[uid] = p
			}
		case "player.protected":
			uid := payload["user_id"]
			if p, ok := state.Players[uid]; ok {
				p.IsProtected = true
				state.Players[uid] = p
			}
		case "demon.changed":
			newDemon := payload["new_demon"]
			state.DemonID = newDemon
			state.ScarletWomanTriggered = true
		}
	}
}
