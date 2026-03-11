package game

import (
	"strings"
	"testing"
)

func TestResolveWasherwomanPoisonedUsesPlausibleFalseInfo(t *testing.T) {
	ctx := &GameContext{
		Players: map[string]*PlayerState{
			"washer":       {UserID: "washer", SeatNumber: 1, TrueRole: "washerwoman", IsAlive: true},
			"investigator": {UserID: "investigator", SeatNumber: 2, TrueRole: "investigator", IsAlive: true},
			"chef":         {UserID: "chef", SeatNumber: 3, TrueRole: "chef", IsAlive: true},
			"imp":          {UserID: "imp", SeatNumber: 4, TrueRole: "imp", IsAlive: true},
		},
		PoisonedIDs: map[string]bool{"washer": true},
	}

	agent := NewNightAgent(ctx)
	result, err := agent.ResolveAbility(AbilityRequest{UserID: "washer", RoleID: "washerwoman", IsFirstNight: true})
	if err != nil {
		t.Fatalf("ResolveAbility returned error: %v", err)
	}
	if result == nil || result.Information == nil {
		t.Fatal("expected poisoned washerwoman to receive false info")
	}
	if !result.Information.IsFalse {
		t.Fatal("expected poisoned washerwoman info to be marked false")
	}
	content, ok := result.Information.Content.(map[string]interface{})
	if !ok {
		t.Fatalf("expected false info content map, got %#v", result.Information.Content)
	}

	players, ok := content["players"].([]string)
	if !ok {
		t.Fatalf("expected players list in false info, got %#v", content["players"])
	}
	if len(players) != 2 {
		t.Fatalf("expected exactly 2 candidate players, got %v", players)
	}
	if content["role"] == "washerwoman" {
		t.Fatalf("expected plausible fake role, got %v", content["role"])
	}
	if content["role"] == "investigator" {
		t.Fatalf("expected poisoned info to avoid reusing the real townsfolk role, got %v", content["role"])
	}
	roleID, ok := content["role"].(string)
	if !ok {
		t.Fatalf("expected fake role id string, got %#v", content["role"])
	}
	role := GetRoleByID(roleID)
	if role == nil || role.Type != RoleTownsfolk {
		t.Fatalf("expected washerwoman false info to stay within townsfolk roles, got %q", roleID)
	}
	if players[0] == "washer" || players[1] == "washer" {
		t.Fatalf("expected false info to exclude the washerwoman player, got %v", players)
	}
	if strings.Contains(result.Message, "Washerwoman") || strings.Contains(result.Message, "Investigator") {
		t.Fatalf("expected role reveal message to use Chinese names, got %q", result.Message)
	}
}

func TestResolveLibrarianPoisonedWithoutOutsiderStillReturnsPairInfo(t *testing.T) {
	ctx := &GameContext{
		Players: map[string]*PlayerState{
			"librarian": {UserID: "librarian", SeatNumber: 1, TrueRole: "librarian", IsAlive: true},
			"chef":      {UserID: "chef", SeatNumber: 2, TrueRole: "chef", IsAlive: true},
			"empath":    {UserID: "empath", SeatNumber: 3, TrueRole: "empath", IsAlive: true},
			"imp":       {UserID: "imp", SeatNumber: 4, TrueRole: "imp", IsAlive: true},
		},
		PoisonedIDs: map[string]bool{"librarian": true},
	}

	agent := NewNightAgent(ctx)
	result, err := agent.ResolveAbility(AbilityRequest{UserID: "librarian", RoleID: "librarian", IsFirstNight: true})
	if err != nil {
		t.Fatalf("ResolveAbility returned error: %v", err)
	}
	if result == nil || result.Information == nil {
		t.Fatal("expected poisoned librarian to receive false info")
	}
	if !result.Information.IsFalse {
		t.Fatal("expected poisoned librarian info to be marked false")
	}
	content, ok := result.Information.Content.(map[string]interface{})
	if !ok {
		t.Fatalf("expected false info content map, got %#v", result.Information.Content)
	}
	if _, hasNoOutsiders := content["no_outsiders"]; hasNoOutsiders {
		t.Fatalf("expected poisoned librarian to receive pair-style false info, got %#v", content)
	}
	players, ok := content["players"].([]string)
	if !ok || len(players) != 2 {
		t.Fatalf("expected two candidate players in false info, got %#v", content["players"])
	}
}

func TestGetRoleDisplayNamePrefersChineseName(t *testing.T) {
	if got := getRoleDisplayName("washerwoman"); got != "洗衣妇" {
		t.Fatalf("expected Chinese role name, got %q", got)
	}
}
