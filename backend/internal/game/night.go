package game

import (
	"fmt"
)

// AbilityResult represents the result of using an ability.
type AbilityResult struct {
	Success     bool            `json:"success"`
	Message     string          `json:"message"`     // Message to show the player
	TrueResult  interface{}     `json:"true_result"` // Actual result (for storyteller)
	FakeResult  interface{}     `json:"fake_result"` // Fake result if poisoned/drunk
	IsPoisoned  bool            `json:"is_poisoned"`
	IsDrunk     bool            `json:"is_drunk"`
	Effects     []AbilityEffect `json:"effects"`
	Information *AbilityInfo    `json:"information,omitempty"`
}

// AbilityEffect represents a side effect of an ability.
type AbilityEffect struct {
	Type      string `json:"type"` // "kill", "poison", "protect", "info"
	TargetID  string `json:"target_id"`
	Value     string `json:"value"`
	ExpiresAt string `json:"expires_at"` // "dawn", "dusk", "never"
}

// AbilityInfo represents information gained from an ability.
type AbilityInfo struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
	IsFalse bool        `json:"is_false"`
}

// AbilityRequest represents a request to use an ability.
type AbilityRequest struct {
	UserID       string   `json:"user_id"`
	RoleID       string   `json:"role_id"`
	TargetIDs    []string `json:"target_ids"`
	ActionType   string   `json:"action_type"` // e.g., "check", "protect", "kill"
	NightNumber  int      `json:"night_number"`
	IsFirstNight bool     `json:"is_first_night"`
}

// GameContext provides game state for ability resolution.
type GameContext struct {
	Players      map[string]*PlayerState
	SeatOrder    []string // UserIDs in seat order
	PoisonedIDs  map[string]bool
	DrunkID      string
	ProtectedIDs map[string]bool
	DeadIDs      map[string]bool
	DemonID      string
	MinionIDs    []string
	NightNumber  int
	RedHerringID string // For fortune teller
}

// PlayerState represents a player's current state.
type PlayerState struct {
	UserID       string
	SeatNumber   int
	Role         string
	TrueRole     string
	Team         Team
	IsAlive      bool
	IsPoisoned   bool
	IsDrunk      bool
	IsProtected  bool
	HasVoted     bool
	HasGhostVote bool
}

// NightAgent handles night ability resolution.
type NightAgent struct {
	ctx *GameContext
}

// NewNightAgent creates a new night agent.
func NewNightAgent(ctx *GameContext) *NightAgent {
	return &NightAgent{ctx: ctx}
}

// ResolveAbility resolves a night ability.
func (na *NightAgent) ResolveAbility(req AbilityRequest) (*AbilityResult, error) {
	player := na.ctx.Players[req.UserID]
	if player == nil {
		return nil, fmt.Errorf("player not found: %s", req.UserID)
	}

	// Check if player is poisoned or drunk
	isPoisoned := na.ctx.PoisonedIDs[req.UserID]
	isDrunk := req.UserID == na.ctx.DrunkID

	malfunctioning := isPoisoned || isDrunk

	switch req.RoleID {
	case "poisoner":
		return na.resolvePoisoner(req, malfunctioning)
	case "spy":
		return na.resolveSpy(req, malfunctioning)
	case "washerwoman":
		return na.resolveWasherwoman(req, malfunctioning)
	case "librarian":
		return na.resolveLibrarian(req, malfunctioning)
	case "investigator":
		return na.resolveInvestigator(req, malfunctioning)
	case "chef":
		return na.resolveChef(req, malfunctioning)
	case "empath":
		return na.resolveEmpath(req, malfunctioning)
	case "fortune_teller":
		return na.resolveFortuneTeller(req, malfunctioning)
	case "undertaker":
		return na.resolveUndertaker(req, malfunctioning)
	case "monk":
		return na.resolveMonk(req, malfunctioning)
	case "ravenkeeper":
		return na.resolveRavenkeeper(req, malfunctioning)
	case "butler":
		return na.resolveButler(req, malfunctioning)
	case "imp":
		return na.resolveImp(req, malfunctioning)
	default:
		return &AbilityResult{
			Success: false,
			Message: fmt.Sprintf("未知角色能力: %s", req.RoleID),
		}, nil
	}
}

// === TOWNSFOLK ABILITIES ===

func (na *NightAgent) resolveWasherwoman(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	if !req.IsFirstNight {
		return &AbilityResult{Success: false, Message: "洗衣妇只在首夜行动"}, nil
	}

	// Find a townsfolk and create the pair
	var townsfolkID, wrongID, townfolkRole string

	for _, p := range na.ctx.Players {
		if p.IsAlive && p.UserID != req.UserID {
			role := GetRoleByID(p.TrueRole)
			if role != nil && role.Type == RoleTownsfolk {
				townsfolkID = p.UserID
				townfolkRole = p.TrueRole
				break
			}
		}
	}

	// Find a wrong player (not the townsfolk)
	for _, p := range na.ctx.Players {
		if p.IsAlive && p.UserID != req.UserID && p.UserID != townsfolkID {
			wrongID = p.UserID
			break
		}
	}

	if townsfolkID == "" || wrongID == "" {
		return &AbilityResult{Success: false, Message: "无法找到足够的玩家"}, nil
	}

	result := &AbilityResult{
		Success:    true,
		IsPoisoned: malfunctioning,
	}

	if malfunctioning {
		// Give false information
		fakeRole := na.getRandomTownsfolkRole(townfolkRole)
		result.Message = fmt.Sprintf("你得知：%s 或 %s 中有一人是 %s",
			na.getPlayerName(townsfolkID), na.getPlayerName(wrongID), getRoleDisplayName(fakeRole))
		result.Information = &AbilityInfo{
			Type:    "washerwoman",
			Content: map[string]interface{}{"players": []string{townsfolkID, wrongID}, "role": fakeRole},
			IsFalse: true,
		}
	} else {
		result.Message = fmt.Sprintf("你得知：%s 或 %s 中有一人是 %s",
			na.getPlayerName(townsfolkID), na.getPlayerName(wrongID), getRoleDisplayName(townfolkRole))
		result.Information = &AbilityInfo{
			Type:    "washerwoman",
			Content: map[string]interface{}{"players": []string{townsfolkID, wrongID}, "role": townfolkRole},
			IsFalse: false,
		}
	}

	return result, nil
}

func (na *NightAgent) resolveLibrarian(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	if !req.IsFirstNight {
		return &AbilityResult{Success: false, Message: "图书管理员只在首夜行动"}, nil
	}

	// Find an outsider
	var outsiderID, wrongID, outsiderRole string

	for _, p := range na.ctx.Players {
		if p.IsAlive && p.UserID != req.UserID {
			role := GetRoleByID(p.TrueRole)
			if role != nil && role.Type == RoleOutsider {
				outsiderID = p.UserID
				outsiderRole = p.TrueRole
				break
			}
		}
	}

	result := &AbilityResult{
		Success:    true,
		IsPoisoned: malfunctioning,
	}

	if outsiderID == "" {
		// No outsiders in play
		if malfunctioning {
			// Might falsely claim there's an outsider
			result.Message = "你得知：场上可能有外来者"
		} else {
			result.Message = "你得知：场上没有外来者"
		}
		result.Information = &AbilityInfo{
			Type:    "librarian",
			Content: map[string]interface{}{"no_outsiders": true},
			IsFalse: malfunctioning,
		}
	} else {
		// Find a wrong player
		for _, p := range na.ctx.Players {
			if p.IsAlive && p.UserID != req.UserID && p.UserID != outsiderID {
				wrongID = p.UserID
				break
			}
		}

		if malfunctioning {
			fakeRole := na.getRandomOutsiderRole(outsiderRole)
			result.Message = fmt.Sprintf("你得知：%s 或 %s 中有一人是 %s",
				na.getPlayerName(outsiderID), na.getPlayerName(wrongID), getRoleDisplayName(fakeRole))
			result.Information = &AbilityInfo{
				Type:    "librarian",
				Content: map[string]interface{}{"players": []string{outsiderID, wrongID}, "role": fakeRole},
				IsFalse: true,
			}
		} else {
			result.Message = fmt.Sprintf("你得知：%s 或 %s 中有一人是 %s",
				na.getPlayerName(outsiderID), na.getPlayerName(wrongID), getRoleDisplayName(outsiderRole))
			result.Information = &AbilityInfo{
				Type:    "librarian",
				Content: map[string]interface{}{"players": []string{outsiderID, wrongID}, "role": outsiderRole},
				IsFalse: false,
			}
		}
	}

	return result, nil
}

func (na *NightAgent) resolveInvestigator(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	if !req.IsFirstNight {
		return &AbilityResult{Success: false, Message: "调查员只在首夜行动"}, nil
	}

	// Find a minion
	var minionID, wrongID, minionRole string

	for _, p := range na.ctx.Players {
		if p.IsAlive && p.UserID != req.UserID {
			role := GetRoleByID(p.TrueRole)
			if role != nil && role.Type == RoleMinion {
				minionID = p.UserID
				minionRole = p.TrueRole
				break
			}
		}
	}

	if minionID == "" {
		return &AbilityResult{Success: false, Message: "无法找到爪牙"}, nil
	}

	// Find a wrong player
	for _, p := range na.ctx.Players {
		if p.IsAlive && p.UserID != req.UserID && p.UserID != minionID {
			wrongID = p.UserID
			break
		}
	}

	result := &AbilityResult{
		Success:    true,
		IsPoisoned: malfunctioning,
	}

	if malfunctioning {
		fakeRole := na.getRandomMinionRole(minionRole)
		result.Message = fmt.Sprintf("你得知：%s 或 %s 中有一人是 %s",
			na.getPlayerName(minionID), na.getPlayerName(wrongID), getRoleDisplayName(fakeRole))
		result.Information = &AbilityInfo{
			Type:    "investigator",
			Content: map[string]interface{}{"players": []string{minionID, wrongID}, "role": fakeRole},
			IsFalse: true,
		}
	} else {
		result.Message = fmt.Sprintf("你得知：%s 或 %s 中有一人是 %s",
			na.getPlayerName(minionID), na.getPlayerName(wrongID), getRoleDisplayName(minionRole))
		result.Information = &AbilityInfo{
			Type:    "investigator",
			Content: map[string]interface{}{"players": []string{minionID, wrongID}, "role": minionRole},
			IsFalse: false,
		}
	}

	return result, nil
}

func (na *NightAgent) resolveChef(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	if !req.IsFirstNight {
		return &AbilityResult{Success: false, Message: "厨师只在首夜行动"}, nil
	}

	// Count evil pairs
	evilPairs := 0
	for i := 0; i < len(na.ctx.SeatOrder); i++ {
		current := na.ctx.SeatOrder[i]
		next := na.ctx.SeatOrder[(i+1)%len(na.ctx.SeatOrder)]

		currentPlayer := na.ctx.Players[current]
		nextPlayer := na.ctx.Players[next]

		if currentPlayer != nil && nextPlayer != nil &&
			currentPlayer.Team == TeamEvil && nextPlayer.Team == TeamEvil {
			evilPairs++
		}
	}

	result := &AbilityResult{
		Success:    true,
		IsPoisoned: malfunctioning,
	}

	if malfunctioning {
		// Give wrong number
		fakePairs := evilPairs
		if evilPairs == 0 {
			fakePairs = 1
		} else {
			fakePairs = 0
		}
		result.Message = fmt.Sprintf("你得知：有 %d 对邪恶玩家彼此相邻", fakePairs)
		result.Information = &AbilityInfo{
			Type:    "chef",
			Content: map[string]interface{}{"pairs": fakePairs},
			IsFalse: true,
		}
	} else {
		result.Message = fmt.Sprintf("你得知：有 %d 对邪恶玩家彼此相邻", evilPairs)
		result.Information = &AbilityInfo{
			Type:    "chef",
			Content: map[string]interface{}{"pairs": evilPairs},
			IsFalse: false,
		}
	}

	return result, nil
}

func (na *NightAgent) resolveEmpath(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	player := na.ctx.Players[req.UserID]
	if player == nil {
		return nil, fmt.Errorf("player not found")
	}

	// Find alive neighbors
	seatIdx := -1
	for i, uid := range na.ctx.SeatOrder {
		if uid == req.UserID {
			seatIdx = i
			break
		}
	}
	if seatIdx == -1 {
		return nil, fmt.Errorf("player not in seat order")
	}

	// Find left alive neighbor
	leftEvil := 0
	for i := 1; i < len(na.ctx.SeatOrder); i++ {
		leftIdx := (seatIdx - i + len(na.ctx.SeatOrder)) % len(na.ctx.SeatOrder)
		leftUID := na.ctx.SeatOrder[leftIdx]
		leftPlayer := na.ctx.Players[leftUID]
		if leftPlayer != nil && leftPlayer.IsAlive {
			if leftPlayer.Team == TeamEvil {
				leftEvil = 1
			}
			break
		}
	}

	// Find right alive neighbor
	rightEvil := 0
	for i := 1; i < len(na.ctx.SeatOrder); i++ {
		rightIdx := (seatIdx + i) % len(na.ctx.SeatOrder)
		rightUID := na.ctx.SeatOrder[rightIdx]
		rightPlayer := na.ctx.Players[rightUID]
		if rightPlayer != nil && rightPlayer.IsAlive {
			if rightPlayer.Team == TeamEvil {
				rightEvil = 1
			}
			break
		}
	}

	evilCount := leftEvil + rightEvil

	result := &AbilityResult{
		Success:    true,
		IsPoisoned: malfunctioning,
	}

	if malfunctioning {
		fakeCount := (evilCount + 1) % 3 // Give wrong number
		result.Message = fmt.Sprintf("你得知：你存活的邻居中有 %d 个邪恶玩家", fakeCount)
		result.Information = &AbilityInfo{
			Type:    "empath",
			Content: map[string]interface{}{"evil_neighbors": fakeCount},
			IsFalse: true,
		}
	} else {
		result.Message = fmt.Sprintf("你得知：你存活的邻居中有 %d 个邪恶玩家", evilCount)
		result.Information = &AbilityInfo{
			Type:    "empath",
			Content: map[string]interface{}{"evil_neighbors": evilCount},
			IsFalse: false,
		}
	}

	return result, nil
}

func (na *NightAgent) resolveFortuneTeller(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	if len(req.TargetIDs) != 2 {
		return &AbilityResult{Success: false, Message: "占卜师需要选择两名玩家"}, nil
	}

	target1 := req.TargetIDs[0]
	target2 := req.TargetIDs[1]

	// Check if either target is the demon
	hasDemon := false
	for _, tid := range req.TargetIDs {
		if tid == na.ctx.DemonID {
			hasDemon = true
			break
		}
		// Check for red herring
		if tid == na.ctx.RedHerringID {
			hasDemon = true
			break
		}
	}

	result := &AbilityResult{
		Success:    true,
		IsPoisoned: malfunctioning,
	}

	if malfunctioning {
		// Give wrong information
		result.Message = fmt.Sprintf("你查验了 %s 和 %s：%s",
			na.getPlayerName(target1), na.getPlayerName(target2),
			formatBool(!hasDemon, "有恶魔", "没有恶魔"))
		result.Information = &AbilityInfo{
			Type:    "fortune_teller",
			Content: map[string]interface{}{"players": req.TargetIDs, "has_demon": !hasDemon},
			IsFalse: true,
		}
	} else {
		result.Message = fmt.Sprintf("你查验了 %s 和 %s：%s",
			na.getPlayerName(target1), na.getPlayerName(target2),
			formatBool(hasDemon, "有恶魔", "没有恶魔"))
		result.Information = &AbilityInfo{
			Type:    "fortune_teller",
			Content: map[string]interface{}{"players": req.TargetIDs, "has_demon": hasDemon},
			IsFalse: false,
		}
	}

	return result, nil
}

func (na *NightAgent) resolveUndertaker(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	if req.IsFirstNight {
		return &AbilityResult{Success: false, Message: "掘墓人不在首夜行动"}, nil
	}

	// This would need the executed player from the day
	// For now, return a placeholder
	result := &AbilityResult{
		Success:    true,
		Message:    "今天没有玩家被处决",
		IsPoisoned: malfunctioning,
	}

	return result, nil
}

func (na *NightAgent) resolveMonk(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	if req.IsFirstNight {
		return &AbilityResult{Success: false, Message: "僧侣不在首夜行动"}, nil
	}

	if len(req.TargetIDs) != 1 {
		return &AbilityResult{Success: false, Message: "僧侣需要选择一名玩家"}, nil
	}

	targetID := req.TargetIDs[0]
	if targetID == req.UserID {
		return &AbilityResult{Success: false, Message: "僧侣不能保护自己"}, nil
	}

	result := &AbilityResult{
		Success:    true,
		Message:    fmt.Sprintf("你保护了 %s", na.getPlayerName(targetID)),
		IsPoisoned: malfunctioning,
	}

	if !malfunctioning {
		result.Effects = append(result.Effects, AbilityEffect{
			Type:      "protect",
			TargetID:  targetID,
			ExpiresAt: "dawn",
		})
	}

	return result, nil
}

func (na *NightAgent) resolveRavenkeeper(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	if len(req.TargetIDs) != 1 {
		return &AbilityResult{Success: false, Message: "守鸦人需要选择一名玩家"}, nil
	}

	targetID := req.TargetIDs[0]
	targetPlayer := na.ctx.Players[targetID]
	if targetPlayer == nil {
		return &AbilityResult{Success: false, Message: "目标玩家不存在"}, nil
	}

	result := &AbilityResult{
		Success:    true,
		IsPoisoned: malfunctioning,
	}

	if malfunctioning {
		fakeRole := na.getRandomRole()
		result.Message = fmt.Sprintf("你得知 %s 的角色是 %s", na.getPlayerName(targetID), getRoleDisplayName(fakeRole))
		result.Information = &AbilityInfo{
			Type:    "ravenkeeper",
			Content: map[string]interface{}{"player": targetID, "role": fakeRole},
			IsFalse: true,
		}
	} else {
		result.Message = fmt.Sprintf("你得知 %s 的角色是 %s", na.getPlayerName(targetID), getRoleDisplayName(targetPlayer.TrueRole))
		result.Information = &AbilityInfo{
			Type:    "ravenkeeper",
			Content: map[string]interface{}{"player": targetID, "role": targetPlayer.TrueRole},
			IsFalse: false,
		}
	}

	return result, nil
}

func (na *NightAgent) resolveButler(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	if len(req.TargetIDs) != 1 {
		return &AbilityResult{Success: false, Message: "管家需要选择一名主人"}, nil
	}

	targetID := req.TargetIDs[0]
	if targetID == req.UserID {
		return &AbilityResult{Success: false, Message: "管家不能选择自己"}, nil
	}

	result := &AbilityResult{
		Success: true,
		Message: fmt.Sprintf("你选择了 %s 作为你的主人", na.getPlayerName(targetID)),
		Effects: []AbilityEffect{{
			Type:      "butler_master",
			TargetID:  targetID,
			ExpiresAt: "dusk",
		}},
	}

	return result, nil
}

// === EVIL ABILITIES ===

func (na *NightAgent) resolvePoisoner(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	if len(req.TargetIDs) != 1 {
		return &AbilityResult{Success: false, Message: "投毒者需要选择一名玩家"}, nil
	}

	targetID := req.TargetIDs[0]

	result := &AbilityResult{
		Success: true,
		Message: fmt.Sprintf("你对 %s 下了毒", na.getPlayerName(targetID)),
	}

	if !malfunctioning {
		result.Effects = append(result.Effects, AbilityEffect{
			Type:      "poison",
			TargetID:  targetID,
			ExpiresAt: "dusk",
		})
	}

	return result, nil
}

func (na *NightAgent) resolveSpy(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	result := &AbilityResult{
		Success: true,
		Message: "你查看了魔典",
	}

	if !malfunctioning {
		// Spy sees all player roles
		grimoire := make(map[string]string)
		for uid, p := range na.ctx.Players {
			grimoire[uid] = p.TrueRole
		}
		result.Information = &AbilityInfo{
			Type:    "spy",
			Content: grimoire,
			IsFalse: false,
		}
	}

	return result, nil
}

func (na *NightAgent) resolveImp(req AbilityRequest, malfunctioning bool) (*AbilityResult, error) {
	if req.IsFirstNight {
		return &AbilityResult{Success: false, Message: "小恶魔不在首夜杀人"}, nil
	}

	if len(req.TargetIDs) != 1 {
		return &AbilityResult{Success: false, Message: "小恶魔需要选择一名玩家"}, nil
	}

	targetID := req.TargetIDs[0]

	// Check if self-kill (starpass)
	isSelfKill := targetID == req.UserID

	result := &AbilityResult{
		Success: true,
	}

	if isSelfKill {
		result.Message = "你选择了自杀，将恶魔身份传给一名爪牙"
		result.Effects = append(result.Effects, AbilityEffect{
			Type:     "starpass",
			TargetID: req.UserID,
		})
	} else {
		// Check if target is protected
		if na.ctx.ProtectedIDs[targetID] {
			// Case 1: Target is protected (e.g. by Monk)
			// The demon should NOT know why the attack failed, so we give a generic message.
			result.Message = fmt.Sprintf("你选择了攻击 %s", na.getPlayerName(targetID))
		} else if na.ctx.Players[targetID] != nil && na.ctx.Players[targetID].TrueRole == "soldier" && !na.ctx.PoisonedIDs[targetID] {
			// Case 2: Target is Soldier (and not poisoned)
			// The demon should NOT know the target is a Soldier, so we give a generic message.
			result.Message = fmt.Sprintf("你选择了攻击 %s", na.getPlayerName(targetID))
		} else {
			// Case 3: Successful attack
			result.Message = fmt.Sprintf("你选择了攻击 %s", na.getPlayerName(targetID))
			result.Effects = append(result.Effects, AbilityEffect{
				Type:     "kill",
				TargetID: targetID,
			})
		}
	}

	return result, nil
}

// === HELPER FUNCTIONS ===

func (na *NightAgent) getPlayerName(userID string) string {
	if p := na.ctx.Players[userID]; p != nil {
		return fmt.Sprintf("玩家%d", p.SeatNumber)
	}
	return "未知玩家"
}

func (na *NightAgent) getRandomTownsfolkRole(exclude string) string {
	roles := GetRolesByType(RoleTownsfolk)
	for _, r := range roles {
		if r.ID != exclude {
			return r.ID
		}
	}
	return "washerwoman"
}

func (na *NightAgent) getRandomOutsiderRole(exclude string) string {
	roles := GetRolesByType(RoleOutsider)
	for _, r := range roles {
		if r.ID != exclude {
			return r.ID
		}
	}
	return "drunk"
}

func (na *NightAgent) getRandomMinionRole(exclude string) string {
	roles := GetRolesByType(RoleMinion)
	for _, r := range roles {
		if r.ID != exclude {
			return r.ID
		}
	}
	return "poisoner"
}

func (na *NightAgent) getRandomRole() string {
	roles := TroubleBrewingRoles
	if len(roles) > 0 {
		idx, _ := randInt(len(roles))
		return roles[idx].ID
	}
	return "villager"
}

func getRoleDisplayName(roleID string) string {
	role := GetRoleByID(roleID)
	if role != nil {
		return role.Name
	}
	return roleID
}

func formatBool(b bool, trueStr, falseStr string) string {
	if b {
		return trueStr
	}
	return falseStr
}
