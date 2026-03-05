// Package game 间谍干扰系统与魔典快照
//
// 提供 getApparentAlignment / getApparentRole 供信息角色使用，
// 以及 GrimoireSnapshot 供间谍查看完整游戏状态。
//
// [OUT] engine（信息分发层调用）
// [POS] 间谍规则实现：间谍对信息角色显示为善良/假身份
package game

// GrimoireSnapshot 间谍查看的完整游戏魔典
type GrimoireSnapshot struct {
	Players []PlayerSnapshot `json:"players"`
}

// PlayerSnapshot 魔典中每位玩家的信息
type PlayerSnapshot struct {
	UserID    string   `json:"user_id"`
	SeatIndex int      `json:"seat_index"`
	Role      string   `json:"role"`      // 真实角色
	Alignment string   `json:"alignment"` // "good" | "evil"
	IsAlive   bool     `json:"is_alive"`
	Poisoned  bool     `json:"poisoned"`
	Protected bool     `json:"protected"`
	Reminders []string `json:"reminders"` // reminder tokens
}

// GetApparentAlignment 返回玩家在信息角色面前的"表观阵营"。
// 间谍对所有信息类角色的阵营查询返回 "good"。
func GetApparentAlignment(playerID string, ctx *GameContext) string {
	p := ctx.Players[playerID]
	if p == nil {
		return "good"
	}
	if p.TrueRole == "spy" {
		return "good"
	}
	if p.Team == TeamEvil {
		return "evil"
	}
	return "good"
}

// GetApparentRole 返回玩家在信息角色面前的"表观角色"。
// 间谍返回其 SpyApparentRole（在 Setup 时由引擎分配的假善良角色）。
func GetApparentRole(playerID string, ctx *GameContext) string {
	p := ctx.Players[playerID]
	if p == nil {
		return ""
	}
	if p.TrueRole == "spy" && p.SpyApparentRole != "" {
		return p.SpyApparentRole
	}
	return p.TrueRole
}

// BuildGrimoireSnapshot 构建间谍查看的完整魔典快照。
// 必须在投毒等效果已经应用后调用，确保反映最终状态。
func BuildGrimoireSnapshot(ctx *GameContext) *GrimoireSnapshot {
	snapshot := &GrimoireSnapshot{
		Players: make([]PlayerSnapshot, 0, len(ctx.SeatOrder)),
	}

	for _, uid := range ctx.SeatOrder {
		p := ctx.Players[uid]
		if p == nil {
			continue
		}
		alignment := "good"
		if p.Team == TeamEvil {
			alignment = "evil"
		}
		var reminders []string
		if ctx.PoisonedIDs[uid] {
			reminders = append(reminders, "中毒")
		}
		if ctx.ProtectedIDs[uid] {
			reminders = append(reminders, "被保护")
		}
		snapshot.Players = append(snapshot.Players, PlayerSnapshot{
			UserID:    uid,
			SeatIndex: p.SeatNumber,
			Role:      p.TrueRole,
			Alignment: alignment,
			IsAlive:   p.IsAlive,
			Poisoned:  ctx.PoisonedIDs[uid],
			Protected: ctx.ProtectedIDs[uid],
			Reminders: reminders,
		})
	}

	return snapshot
}
