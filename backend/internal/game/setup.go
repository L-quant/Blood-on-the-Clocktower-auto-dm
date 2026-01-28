package game

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// Edition represents a game edition.
type Edition string

const (
	EditionTroubleBrewing Edition = "tb"
	EditionBadMoonRising  Edition = "bmr"
	EditionSectsAndViolet Edition = "snv"
)

// Script represents a game script (edition) configuration.
type Script struct {
	Edition   Edition
	RolesPool []Role
}

// SetupConfig holds configuration for game setup.
type SetupConfig struct {
	Script      *Script
	PlayerCount int
	CustomRoles []string // Override automatic role selection
	BaronActive bool     // Add +2 outsiders
	DrunkTarget string   // Role that drunk thinks they are
}

// SetupResult holds the result of role assignment.
type SetupResult struct {
	Assignments   map[string]Assignment // UserID -> Assignment
	BluffRoles    []string              // 3 roles not in play for demon
	NightOrder    []NightAction         // First night wake order
	DrunkRole     string                // What role the drunk thinks they are
	BaronModified bool                  // Whether baron modified outsider count
}

// Assignment represents a player's assigned role.
type Assignment struct {
	UserID        string   `json:"user_id"`
	SeatNumber    int      `json:"seat_number"`
	Role          string   `json:"role"`
	TrueRole      string   `json:"true_role"`      // For drunk: actual role
	PerceivedRole string   `json:"perceived_role"` // For drunk: what they think
	Team          Team     `json:"team"`
	Teammates     []string `json:"teammates,omitempty"` // For evil team
	DemonID       string   `json:"demon_id,omitempty"`  // For minions
}

// NightAction represents a night wake action.
type NightAction struct {
	Order    int    `json:"order"`
	RoleID   string `json:"role_id"`
	RoleName string `json:"role_name"`
	UserID   string `json:"user_id"`
	Action   string `json:"action"`
}

// SetupAgent handles game setup logic.
type SetupAgent struct {
	config SetupConfig
}

// NewSetupAgent creates a new setup agent.
func NewSetupAgent(config SetupConfig) *SetupAgent {
	return &SetupAgent{config: config}
}

// GenerateAssignments creates role assignments for all players.
func (sa *SetupAgent) GenerateAssignments(userIDs []string, seatOrder []int) (*SetupResult, error) {
	playerCount := len(userIDs)
	if playerCount < 5 || playerCount > 15 {
		return nil, fmt.Errorf("player count must be between 5 and 15, got %d", playerCount)
	}

	dist := GetDistribution(playerCount)
	if dist == nil {
		return nil, fmt.Errorf("no distribution for %d players", playerCount)
	}

	// Adjust for Baron
	outsiderCount := dist.Outsiders
	townsfolkCount := dist.Townsfolk

	// Check if Baron is in the script and will be assigned
	baronInPlay := false
	if sa.config.BaronActive {
		baronInPlay = true
		outsiderCount += 2
		townsfolkCount -= 2
		if townsfolkCount < 0 {
			townsfolkCount = 0
		}
	}

	// Get available roles by type
	availableTownsfolk := GetRolesByType(RoleTownsfolk)
	availableOutsiders := GetRolesByType(RoleOutsider)
	availableMinions := GetRolesByType(RoleMinion)
	availableDemons := GetRolesByType(RoleDemon)

	// Select roles
	selectedRoles := make([]Role, 0, playerCount)

	// Select demon(s)
	demons, err := selectRandomRoles(availableDemons, dist.Demons)
	if err != nil {
		return nil, fmt.Errorf("selecting demons: %w", err)
	}
	selectedRoles = append(selectedRoles, demons...)

	// Select minions
	minions, err := selectRandomRoles(availableMinions, dist.Minions)
	if err != nil {
		return nil, fmt.Errorf("selecting minions: %w", err)
	}
	selectedRoles = append(selectedRoles, minions...)

	// Select outsiders
	outsiders, err := selectRandomRoles(availableOutsiders, outsiderCount)
	if err != nil {
		return nil, fmt.Errorf("selecting outsiders: %w", err)
	}
	selectedRoles = append(selectedRoles, outsiders...)

	// Select townsfolk (fill remaining slots)
	remainingSlots := playerCount - len(selectedRoles)
	townsfolk, err := selectRandomRoles(availableTownsfolk, remainingSlots)
	if err != nil {
		return nil, fmt.Errorf("selecting townsfolk: %w", err)
	}
	selectedRoles = append(selectedRoles, townsfolk...)

	// Shuffle selected roles
	shuffledRoles, err := shuffleRoles(selectedRoles)
	if err != nil {
		return nil, fmt.Errorf("shuffling roles: %w", err)
	}

	// Create assignments
	assignments := make(map[string]Assignment)
	var demonID string
	var minionIDs []string

	// First pass: identify evil team
	for i, userID := range userIDs {
		role := shuffledRoles[i]
		if role.Type == RoleDemon {
			demonID = userID
		} else if role.Type == RoleMinion {
			minionIDs = append(minionIDs, userID)
		}
	}

	// Handle drunk role
	drunkRole := ""
	for i, role := range shuffledRoles {
		if role.ID == "drunk" {
			_ = userIDs[i] // drunkUserID - could be used for additional tracking
			// Select a random townsfolk role for drunk to "think" they are
			if len(availableTownsfolk) > 0 {
				drunkIdx, _ := randInt(len(availableTownsfolk))
				drunkRole = availableTownsfolk[drunkIdx].ID
			}
			break
		}
	}

	// Second pass: create assignments
	for i, userID := range userIDs {
		role := shuffledRoles[i]
		seatNum := i + 1
		if len(seatOrder) > i {
			seatNum = seatOrder[i]
		}

		assignment := Assignment{
			UserID:        userID,
			SeatNumber:    seatNum,
			Role:          role.ID,
			TrueRole:      role.ID,
			PerceivedRole: role.ID,
			Team:          role.Team,
		}

		// Handle drunk
		if role.ID == "drunk" && drunkRole != "" {
			assignment.PerceivedRole = drunkRole
		}

		// Evil team info
		if role.Team == TeamEvil {
			var teammates []string
			if role.Type == RoleDemon {
				teammates = minionIDs
			} else {
				teammates = append(teammates, demonID)
				for _, mid := range minionIDs {
					if mid != userID {
						teammates = append(teammates, mid)
					}
				}
			}
			assignment.Teammates = teammates
			assignment.DemonID = demonID
		}

		assignments[userID] = assignment
	}

	// Generate bluff roles (3 roles not in play for demon)
	bluffRoles := generateBluffs(shuffledRoles, availableTownsfolk, availableOutsiders)

	// Generate first night order
	nightOrder := generateNightOrder(shuffledRoles, assignments, true)

	return &SetupResult{
		Assignments:   assignments,
		BluffRoles:    bluffRoles,
		NightOrder:    nightOrder,
		DrunkRole:     drunkRole,
		BaronModified: baronInPlay,
	}, nil
}

// selectRandomRoles selects n random roles from the available pool.
func selectRandomRoles(pool []Role, count int) ([]Role, error) {
	if count > len(pool) {
		count = len(pool)
	}
	if count == 0 {
		return nil, nil
	}

	// Create copy of pool
	poolCopy := make([]Role, len(pool))
	copy(poolCopy, pool)

	selected := make([]Role, 0, count)
	for i := 0; i < count; i++ {
		idx, err := randInt(len(poolCopy))
		if err != nil {
			return nil, err
		}
		selected = append(selected, poolCopy[idx])
		// Remove selected role
		poolCopy = append(poolCopy[:idx], poolCopy[idx+1:]...)
	}
	return selected, nil
}

// shuffleRoles shuffles the role slice randomly.
func shuffleRoles(roles []Role) ([]Role, error) {
	shuffled := make([]Role, len(roles))
	copy(shuffled, roles)

	for i := len(shuffled) - 1; i > 0; i-- {
		j, err := randInt(i + 1)
		if err != nil {
			return nil, err
		}
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled, nil
}

// randInt returns a random int in [0, n).
func randInt(n int) (int, error) {
	if n <= 0 {
		return 0, nil
	}
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(nBig.Int64()), nil
}

// generateBluffs generates 3 safe bluff roles for the demon.
func generateBluffs(inPlay []Role, townsfolk, outsiders []Role) []string {
	inPlayIDs := make(map[string]bool)
	for _, r := range inPlay {
		inPlayIDs[r.ID] = true
	}

	var candidates []string
	for _, r := range townsfolk {
		if !inPlayIDs[r.ID] {
			candidates = append(candidates, r.ID)
		}
	}
	for _, r := range outsiders {
		if !inPlayIDs[r.ID] {
			candidates = append(candidates, r.ID)
		}
	}

	// Select 3 random bluffs
	var bluffs []string
	for i := 0; i < 3 && len(candidates) > 0; i++ {
		idx, _ := randInt(len(candidates))
		bluffs = append(bluffs, candidates[idx])
		candidates = append(candidates[:idx], candidates[idx+1:]...)
	}
	return bluffs
}

// generateNightOrder generates the night wake order.
func generateNightOrder(roles []Role, assignments map[string]Assignment, firstNight bool) []NightAction {
	type orderedRole struct {
		role   Role
		userID string
		order  int
	}

	var ordered []orderedRole
	for userID, assignment := range assignments {
		role := GetRoleByID(assignment.TrueRole)
		if role == nil {
			continue
		}

		var order int
		if firstNight {
			order = role.FirstNightOrder
		} else {
			order = role.OtherNightOrder
		}

		if order > 0 {
			ordered = append(ordered, orderedRole{
				role:   *role,
				userID: userID,
				order:  order,
			})
		}
	}

	// Sort by order
	for i := 0; i < len(ordered)-1; i++ {
		for j := i + 1; j < len(ordered); j++ {
			if ordered[i].order > ordered[j].order {
				ordered[i], ordered[j] = ordered[j], ordered[i]
			}
		}
	}

	var actions []NightAction
	for i, or := range ordered {
		actions = append(actions, NightAction{
			Order:    i + 1,
			RoleID:   or.role.ID,
			RoleName: or.role.Name,
			UserID:   or.userID,
			Action:   describeNightAction(or.role, firstNight),
		})
	}
	return actions
}

// describeNightAction returns a description of the night action.
func describeNightAction(role Role, firstNight bool) string {
	switch role.ID {
	case "poisoner":
		return "选择一名玩家进行投毒"
	case "spy":
		return "查看魔典"
	case "washerwoman":
		return "得知两名玩家中谁是特定镇民"
	case "librarian":
		return "得知两名玩家中谁是特定外来者"
	case "investigator":
		return "得知两名玩家中谁是特定爪牙"
	case "chef":
		return "得知有多少对邪恶玩家相邻"
	case "empath":
		return "得知存活邻居中有多少邪恶玩家"
	case "fortuneteller":
		return "选择两名玩家，得知是否有恶魔"
	case "undertaker":
		return "得知今天被处决玩家的角色"
	case "monk":
		return "选择一名玩家保护其免受恶魔杀害"
	case "ravenkeeper":
		return "如果死亡，选择一名玩家得知其角色"
	case "butler":
		return "选择你的主人"
	case "imp":
		return "选择一名玩家杀死"
	default:
		return "执行夜间能力"
	}
}
