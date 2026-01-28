// Package game provides core game mechanics for Blood on the Clocktower.
package game

// Team represents the team/alignment of a role.
type Team string

const (
	TeamGood Team = "good"
	TeamEvil Team = "evil"
)

// RoleType represents the type of role.
type RoleType string

const (
	RoleTownsfolk RoleType = "townsfolk"
	RoleOutsider  RoleType = "outsider"
	RoleMinion    RoleType = "minion"
	RoleDemon     RoleType = "demon"
)

// AbilityType represents when an ability activates.
type AbilityType string

const (
	AbilityPassive    AbilityType = "passive"
	AbilityFirstNight AbilityType = "first_night"
	AbilityNight      AbilityType = "night"
	AbilityDay        AbilityType = "day"
	AbilityOnDeath    AbilityType = "on_death"
)

// Role represents a game role with all its properties.
type Role struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	NameCN          string      `json:"name_cn"`
	Team            Team        `json:"team"`
	Type            RoleType    `json:"type"`
	Ability         string      `json:"ability"`
	AbilityCN       string      `json:"ability_cn"`
	AbilityType     AbilityType `json:"ability_type"`
	FirstNightOrder int         `json:"first_night_order"`
	OtherNightOrder int         `json:"other_night_order"`
	Reminders       []string    `json:"reminders"`
	Setup           bool        `json:"setup"`
}

// TroubleBrewingRoles contains all Trouble Brewing edition roles.
var TroubleBrewingRoles = []Role{
	// Townsfolk
	{ID: "washerwoman", Name: "Washerwoman", NameCN: "洗衣妇", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityFirstNight, FirstNightOrder: 32, Ability: "You start knowing that 1 of 2 players is a particular Townsfolk.", AbilityCN: "你在首个夜晚会得知2名玩家中有1名是特定的村民。"},
	{ID: "librarian", Name: "Librarian", NameCN: "图书管理员", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityFirstNight, FirstNightOrder: 33, Ability: "You start knowing that 1 of 2 players is a particular Outsider. (Or that zero are in play.)", AbilityCN: "你在首个夜晚会得知2名玩家中有1名是特定的外来者，或得知场上没有外来者。"},
	{ID: "investigator", Name: "Investigator", NameCN: "调查员", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityFirstNight, FirstNightOrder: 34, Ability: "You start knowing that 1 of 2 players is a particular Minion.", AbilityCN: "你在首个夜晚会得知2名玩家中有1名是特定的爪牙。"},
	{ID: "chef", Name: "Chef", NameCN: "厨师", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityFirstNight, FirstNightOrder: 35, Ability: "You start knowing how many pairs of evil players there are.", AbilityCN: "你在首个夜晚会得知场上有多少对相邻的邪恶玩家。"},
	{ID: "empath", Name: "Empath", NameCN: "共情者", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityNight, FirstNightOrder: 36, OtherNightOrder: 53, Ability: "Each night, you learn how many of your 2 alive neighbours are evil.", AbilityCN: "每个夜晚，你会得知你两侧存活的邻居中有多少名是邪恶的。"},
	{ID: "fortune_teller", Name: "Fortune Teller", NameCN: "占卜师", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityNight, FirstNightOrder: 37, OtherNightOrder: 54, Ability: "Each night, choose 2 players: you learn if either is a Demon. There is a good player that registers as a Demon to you.", AbilityCN: "每个夜晚，选择2名玩家：你会得知他们中是否有恶魔。有一名善良玩家会被你探测为恶魔。", Reminders: []string{"Red herring"}},
	{ID: "undertaker", Name: "Undertaker", NameCN: "掘墓人", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityNight, OtherNightOrder: 55, Ability: "Each night*, you learn which character died by execution today.", AbilityCN: "每个夜晚*，你会得知今天被处决的玩家的角色。"},
	{ID: "monk", Name: "Monk", NameCN: "僧侣", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityNight, OtherNightOrder: 12, Ability: "Each night*, choose a player (not yourself): they are safe from the Demon tonight.", AbilityCN: "每个夜晚*，选择一名玩家（非自己）：他们今晚免受恶魔伤害。", Reminders: []string{"Protected"}},
	{ID: "ravenkeeper", Name: "Ravenkeeper", NameCN: "守鸦人", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityOnDeath, Ability: "If you die at night, you are woken to choose a player: you learn their character.", AbilityCN: "如果你在夜晚死亡，你会被唤醒并选择一名玩家：你会得知他们的角色。"},
	{ID: "virgin", Name: "Virgin", NameCN: "贞洁者", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityPassive, Ability: "The 1st time you are nominated, if the nominator is a Townsfolk, they are executed immediately.", AbilityCN: "你第一次被提名时，如果提名者是村民，他们会被立即处决。", Reminders: []string{"No ability"}},
	{ID: "slayer", Name: "Slayer", NameCN: "杀手", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityDay, Ability: "Once per game, during the day, publicly choose a player: if they are the Demon, they die.", AbilityCN: "游戏中一次，在白天，公开选择一名玩家：如果他们是恶魔，他们死亡。", Reminders: []string{"No ability"}},
	{ID: "soldier", Name: "Soldier", NameCN: "士兵", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityPassive, Ability: "You are safe from the Demon.", AbilityCN: "你免受恶魔的伤害。"},
	{ID: "mayor", Name: "Mayor", NameCN: "市长", Team: TeamGood, Type: RoleTownsfolk, AbilityType: AbilityPassive, Ability: "If only 3 players live & no execution occurs, your team wins. If you die at night, another player might die instead.", AbilityCN: "如果只剩3名玩家存活且没有处决发生，你的阵营获胜。如果你在夜晚死亡，另一名玩家可能代替你死亡。"},

	// Outsiders
	{ID: "butler", Name: "Butler", NameCN: "管家", Team: TeamGood, Type: RoleOutsider, AbilityType: AbilityNight, FirstNightOrder: 38, OtherNightOrder: 56, Ability: "Each night, choose a player (not yourself): tomorrow, you may only vote if they are voting too.", AbilityCN: "每个夜晚，选择一名玩家（非自己）：明天，只有当他们投票时你才能投票。", Reminders: []string{"Master"}},
	{ID: "drunk", Name: "Drunk", NameCN: "酒鬼", Team: TeamGood, Type: RoleOutsider, AbilityType: AbilityPassive, Setup: true, Ability: "You do not know you are the Drunk. You think you are a Townsfolk character, but you are not.", AbilityCN: "你不知道自己是酒鬼。你认为自己是一名村民角色，但你不是。"},
	{ID: "recluse", Name: "Recluse", NameCN: "隐士", Team: TeamGood, Type: RoleOutsider, AbilityType: AbilityPassive, Ability: "You might register as evil & as a Minion or Demon, even if dead.", AbilityCN: "你可能被探测为邪恶阵营、爪牙或恶魔，即使你已经死亡。"},
	{ID: "saint", Name: "Saint", NameCN: "圣徒", Team: TeamGood, Type: RoleOutsider, AbilityType: AbilityPassive, Ability: "If you die by execution, your team loses.", AbilityCN: "如果你被处决致死，你的阵营失败。"},

	// Minions
	{ID: "poisoner", Name: "Poisoner", NameCN: "投毒者", Team: TeamEvil, Type: RoleMinion, AbilityType: AbilityNight, FirstNightOrder: 17, OtherNightOrder: 7, Ability: "Each night, choose a player: they are poisoned tonight and tomorrow day.", AbilityCN: "每个夜晚，选择一名玩家：他们今晚和明天白天中毒。", Reminders: []string{"Poisoned"}},
	{ID: "spy", Name: "Spy", NameCN: "间谍", Team: TeamEvil, Type: RoleMinion, AbilityType: AbilityNight, FirstNightOrder: 49, OtherNightOrder: 68, Ability: "Each night, you see the Grimoire. You might register as good & as a Townsfolk or Outsider, even if dead.", AbilityCN: "每个夜晚，你可以查看魔典。你可能被探测为善良阵营、村民或外来者，即使你已经死亡。"},
	{ID: "scarlet_woman", Name: "Scarlet Woman", NameCN: "红衣女郎", Team: TeamEvil, Type: RoleMinion, AbilityType: AbilityPassive, Ability: "If there are 5 or more players alive & the Demon dies, you become the Demon. (Travellers don't count)", AbilityCN: "如果场上有5名或更多玩家存活且恶魔死亡，你将成为恶魔。（旅行者不计入）"},
	{ID: "baron", Name: "Baron", NameCN: "男爵", Team: TeamEvil, Type: RoleMinion, AbilityType: AbilityPassive, Setup: true, Ability: "There are extra Outsiders in play. [+2 Outsiders]", AbilityCN: "场上有额外的外来者。[+2 外来者]"},

	// Demon
	{ID: "imp", Name: "Imp", NameCN: "小恶魔", Team: TeamEvil, Type: RoleDemon, AbilityType: AbilityNight, FirstNightOrder: 25, OtherNightOrder: 24, Ability: "Each night*, choose a player: they die. If you kill yourself this way, a Minion becomes the Imp.", AbilityCN: "每个夜晚*，选择一名玩家：他们死亡。如果你用这种方式杀死自己，一名爪牙将成为小恶魔。", Reminders: []string{"Dead"}},
}

// PlayerDistribution defines how many of each role type for a given player count.
type PlayerDistribution struct {
	PlayerCount int
	Townsfolk   int
	Outsiders   int
	Minions     int
	Demons      int
}

// TroubleBrewingDistributions defines player distributions for Trouble Brewing.
var TroubleBrewingDistributions = []PlayerDistribution{
	{PlayerCount: 5, Townsfolk: 3, Outsiders: 0, Minions: 1, Demons: 1},
	{PlayerCount: 6, Townsfolk: 3, Outsiders: 1, Minions: 1, Demons: 1},
	{PlayerCount: 7, Townsfolk: 5, Outsiders: 0, Minions: 1, Demons: 1},
	{PlayerCount: 8, Townsfolk: 5, Outsiders: 1, Minions: 1, Demons: 1},
	{PlayerCount: 9, Townsfolk: 5, Outsiders: 2, Minions: 1, Demons: 1},
	{PlayerCount: 10, Townsfolk: 7, Outsiders: 0, Minions: 2, Demons: 1},
	{PlayerCount: 11, Townsfolk: 7, Outsiders: 1, Minions: 2, Demons: 1},
	{PlayerCount: 12, Townsfolk: 7, Outsiders: 2, Minions: 2, Demons: 1},
	{PlayerCount: 13, Townsfolk: 9, Outsiders: 0, Minions: 3, Demons: 1},
	{PlayerCount: 14, Townsfolk: 9, Outsiders: 1, Minions: 3, Demons: 1},
	{PlayerCount: 15, Townsfolk: 9, Outsiders: 2, Minions: 3, Demons: 1},
}

var roleMap map[string]*Role

func init() {
	roleMap = make(map[string]*Role)
	for i := range TroubleBrewingRoles {
		roleMap[TroubleBrewingRoles[i].ID] = &TroubleBrewingRoles[i]
	}
}

// GetRoleByID returns a role by its ID.
func GetRoleByID(id string) *Role {
	return roleMap[id]
}

// GetRolesByType returns all roles of a given type.
func GetRolesByType(roleType RoleType) []Role {
	var roles []Role
	for _, r := range TroubleBrewingRoles {
		if r.Type == roleType {
			roles = append(roles, r)
		}
	}
	return roles
}

// GetDistribution returns the player distribution for a given player count.
func GetDistribution(playerCount int) *PlayerDistribution {
	for _, d := range TroubleBrewingDistributions {
		if d.PlayerCount == playerCount {
			return &d
		}
	}
	return nil
}

// GetNightOrder returns the night order for all roles.
func GetNightOrder(firstNight bool) []Role {
	var roles []Role
	for _, r := range TroubleBrewingRoles {
		if firstNight && r.FirstNightOrder > 0 {
			roles = append(roles, r)
		} else if !firstNight && r.OtherNightOrder > 0 {
			roles = append(roles, r)
		}
	}
	// Sort by order
	for i := 0; i < len(roles)-1; i++ {
		for j := i + 1; j < len(roles); j++ {
			orderI := roles[i].FirstNightOrder
			orderJ := roles[j].FirstNightOrder
			if !firstNight {
				orderI = roles[i].OtherNightOrder
				orderJ = roles[j].OtherNightOrder
			}
			if orderI > orderJ {
				roles[i], roles[j] = roles[j], roles[i]
			}
		}
	}
	return roles
}
