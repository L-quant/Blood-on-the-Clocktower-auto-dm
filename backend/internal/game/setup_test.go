package game

import (
	"context"
	"fmt"
	"testing"
)

func TestBaronAutoDetection(t *testing.T) {
	// Force a game with baron by using CustomRoles
	customRoles := []string{"imp", "baron", "washerwoman", "librarian", "chef", "drunk", "recluse"}
	config := SetupConfig{
		PlayerCount: 7,
		CustomRoles: customRoles,
	}
	sa := NewSetupAgent(config)
	userIDs := []string{"u1", "u2", "u3", "u4", "u5", "u6", "u7"}
	result, err := sa.GenerateAssignments(userIDs, nil)
	if err != nil {
		t.Fatalf("GenerateAssignments failed: %v", err)
	}
	if !result.BaronModified {
		t.Error("expected BaronModified=true when baron is in CustomRoles")
	}
}

func TestCustomRolesUsed(t *testing.T) {
	customRoles := []string{"imp", "poisoner", "washerwoman", "empath", "fortuneteller", "butler", "saint"}
	config := SetupConfig{
		PlayerCount: 7,
		CustomRoles: customRoles,
	}
	sa := NewSetupAgent(config)
	userIDs := []string{"u1", "u2", "u3", "u4", "u5", "u6", "u7"}
	result, err := sa.GenerateAssignments(userIDs, nil)
	if err != nil {
		t.Fatalf("GenerateAssignments failed: %v", err)
	}

	// Verify all custom roles are assigned
	assigned := make(map[string]bool)
	for _, a := range result.Assignments {
		assigned[a.Role] = true
	}
	for _, roleID := range customRoles {
		if !assigned[roleID] {
			t.Errorf("custom role %s was not assigned", roleID)
		}
	}
}

func TestCustomRolesCountMismatch(t *testing.T) {
	config := SetupConfig{
		PlayerCount: 7,
		CustomRoles: []string{"imp", "poisoner", "washerwoman"}, // Only 3, need 7
	}
	sa := NewSetupAgent(config)
	userIDs := []string{"u1", "u2", "u3", "u4", "u5", "u6", "u7"}
	_, err := sa.GenerateAssignments(userIDs, nil)
	if err == nil {
		t.Error("expected error for mismatched custom roles count")
	}
}

func TestCustomRolesUnknownRole(t *testing.T) {
	config := SetupConfig{
		PlayerCount: 5,
		CustomRoles: []string{"imp", "poisoner", "nonexistent_role", "empath", "chef"},
	}
	sa := NewSetupAgent(config)
	userIDs := []string{"u1", "u2", "u3", "u4", "u5"}
	_, err := sa.GenerateAssignments(userIDs, nil)
	if err == nil {
		t.Error("expected error for unknown role ID")
	}
}

func TestRandomComposer(t *testing.T) {
	rc := &RandomComposer{}
	result, err := rc.ComposeRoles(context.Background(), ComposeRequest{
		PlayerCount: 7,
		Edition:     "tb",
	})
	if err != nil {
		t.Fatalf("ComposeRoles failed: %v", err)
	}
	if len(result.Roles) != 7 {
		t.Errorf("expected 7 roles, got %d", len(result.Roles))
	}
	// Verify all role IDs are valid
	for _, id := range result.Roles {
		if GetRoleByID(id) == nil {
			t.Errorf("unknown role ID: %s", id)
		}
	}
}

func TestFallbackComposer(t *testing.T) {
	failing := &failingComposer{}
	random := &RandomComposer{}
	fc := &FallbackComposer{Primary: failing, Fallback: random}

	result, err := fc.ComposeRoles(context.Background(), ComposeRequest{
		PlayerCount: 7,
		Edition:     "tb",
	})
	if err != nil {
		t.Fatalf("FallbackComposer should succeed with fallback: %v", err)
	}
	if len(result.Roles) != 7 {
		t.Errorf("expected 7 roles, got %d", len(result.Roles))
	}
	if result.Reasoning != "fallback: random selection" {
		t.Errorf("expected fallback reasoning, got %q", result.Reasoning)
	}
}

func TestBaronRandomSelection(t *testing.T) {
	// Run random selection many times to verify Baron adjustment works
	// when Baron is randomly selected
	dist := GetDistribution(7)
	if dist == nil {
		t.Fatal("no distribution for 7 players")
	}

	baronFound := false
	for i := 0; i < 100; i++ {
		roles, baronInPlay, err := selectRolesRandomly(dist, 7)
		if err != nil {
			t.Fatalf("selectRolesRandomly failed: %v", err)
		}
		if baronInPlay {
			baronFound = true
			// When Baron is in play, should have +2 outsiders
			outsiderCount := 0
			for _, r := range roles {
				if r.Type == RoleOutsider {
					outsiderCount++
				}
			}
			// 7-player base has 0 outsiders, +2 for Baron = 2
			if outsiderCount != 2 {
				t.Errorf("with Baron: expected 2 outsiders, got %d", outsiderCount)
			}
		}
	}
	// Baron should appear at least once in 100 tries (probability is high)
	if !baronFound {
		t.Log("Baron was never randomly selected in 100 tries (unlikely but possible)")
	}
}

type failingComposer struct{}

func (fc *failingComposer) ComposeRoles(_ context.Context, _ ComposeRequest) (*ComposeResult, error) {
	return nil, fmt.Errorf("intentional failure")
}
