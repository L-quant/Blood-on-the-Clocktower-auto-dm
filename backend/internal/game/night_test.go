package game

import (
	"testing"
)

// buildTestContext creates a standard 7-player test context.
func buildTestContext() *GameContext {
	return &GameContext{
		Players: map[string]*PlayerState{
			"washer":  {UserID: "washer", SeatNumber: 1, Role: "washerwoman", TrueRole: "washerwoman", Team: TeamGood, IsAlive: true},
			"empath":  {UserID: "empath", SeatNumber: 2, Role: "empath", TrueRole: "empath", Team: TeamGood, IsAlive: true},
			"fort":    {UserID: "fort", SeatNumber: 3, Role: "fortune_teller", TrueRole: "fortune_teller", Team: TeamGood, IsAlive: true},
			"monk":    {UserID: "monk", SeatNumber: 4, Role: "monk", TrueRole: "monk", Team: TeamGood, IsAlive: true},
			"butler":  {UserID: "butler", SeatNumber: 5, Role: "butler", TrueRole: "butler", Team: TeamGood, IsAlive: true},
			"poison":  {UserID: "poison", SeatNumber: 6, Role: "poisoner", TrueRole: "poisoner", Team: TeamEvil, IsAlive: true},
			"imp":     {UserID: "imp", SeatNumber: 7, Role: "imp", TrueRole: "imp", Team: TeamEvil, IsAlive: true},
		},
		SeatOrder:   []string{"washer", "empath", "fort", "monk", "butler", "poison", "imp"},
		PoisonedIDs: make(map[string]bool),
		ProtectedIDs: make(map[string]bool),
		DeadIDs:     make(map[string]bool),
		DemonID:     "imp",
		MinionIDs:   []string{"poison"},
		NightNumber: 1,
	}
}

func TestImpKillNormal(t *testing.T) {
	ctx := buildTestContext()
	ctx.NightNumber = 2
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:      "imp",
		RoleID:      "imp",
		TargetIDs:   []string{"washer"},
		NightNumber: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Message)
	}
	// Should have a kill effect
	hasKill := false
	for _, e := range result.Effects {
		if e.Type == "kill" && e.TargetID == "washer" {
			hasKill = true
		}
	}
	if !hasKill {
		t.Errorf("expected kill effect on washer")
	}
}

func TestImpKillSoldierImmunity(t *testing.T) {
	ctx := buildTestContext()
	ctx.NightNumber = 2
	ctx.Players["soldier"] = &PlayerState{UserID: "soldier", SeatNumber: 8, TrueRole: "soldier", Team: TeamGood, IsAlive: true}
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:      "imp",
		RoleID:      "imp",
		TargetIDs:   []string{"soldier"},
		NightNumber: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Soldier should not have a kill effect
	for _, e := range result.Effects {
		if e.Type == "kill" {
			t.Errorf("soldier should not be killed")
		}
	}
}

func TestImpKillPoisonedSoldierDies(t *testing.T) {
	ctx := buildTestContext()
	ctx.NightNumber = 2
	ctx.Players["soldier"] = &PlayerState{UserID: "soldier", SeatNumber: 8, TrueRole: "soldier", Team: TeamGood, IsAlive: true}
	ctx.PoisonedIDs["soldier"] = true
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:      "imp",
		RoleID:      "imp",
		TargetIDs:   []string{"soldier"},
		NightNumber: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	hasKill := false
	for _, e := range result.Effects {
		if e.Type == "kill" && e.TargetID == "soldier" {
			hasKill = true
		}
	}
	if !hasKill {
		t.Errorf("poisoned soldier should be killed")
	}
}

func TestImpKillMonkProtected(t *testing.T) {
	ctx := buildTestContext()
	ctx.NightNumber = 2
	ctx.ProtectedIDs["washer"] = true
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:      "imp",
		RoleID:      "imp",
		TargetIDs:   []string{"washer"},
		NightNumber: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range result.Effects {
		if e.Type == "kill" {
			t.Errorf("monk-protected player should not be killed")
		}
	}
}

func TestMayorDeathBounce(t *testing.T) {
	ctx := buildTestContext()
	ctx.NightNumber = 2
	ctx.Players["mayor"] = &PlayerState{UserID: "mayor", SeatNumber: 8, TrueRole: "mayor", Team: TeamGood, IsAlive: true}
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:      "imp",
		RoleID:      "imp",
		TargetIDs:   []string{"mayor"},
		NightNumber: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Should have a kill effect but NOT on the mayor
	for _, e := range result.Effects {
		if e.Type == "kill" && e.TargetID == "mayor" {
			t.Errorf("mayor should not be the kill target (death bounces)")
		}
		if e.Type == "kill" && e.TargetID == "imp" {
			t.Errorf("kill should not bounce to demon")
		}
	}
	// Should have exactly one kill effect on someone else
	killCount := 0
	for _, e := range result.Effects {
		if e.Type == "kill" {
			killCount++
		}
	}
	if killCount != 1 {
		t.Errorf("expected exactly 1 kill effect (bounce), got %d", killCount)
	}
}

func TestPoisonedMayorNoBounce(t *testing.T) {
	ctx := buildTestContext()
	ctx.NightNumber = 2
	ctx.Players["mayor"] = &PlayerState{UserID: "mayor", SeatNumber: 8, TrueRole: "mayor", Team: TeamGood, IsAlive: true}
	ctx.PoisonedIDs["mayor"] = true
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:      "imp",
		RoleID:      "imp",
		TargetIDs:   []string{"mayor"},
		NightNumber: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Poisoned mayor should die normally
	hasKillOnMayor := false
	for _, e := range result.Effects {
		if e.Type == "kill" && e.TargetID == "mayor" {
			hasKillOnMayor = true
		}
	}
	if !hasKillOnMayor {
		t.Errorf("poisoned mayor should die normally")
	}
}

func TestStarpassCreatesEffect(t *testing.T) {
	ctx := buildTestContext()
	ctx.NightNumber = 2
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:      "imp",
		RoleID:      "imp",
		TargetIDs:   []string{"imp"}, // Self-kill
		NightNumber: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	hasStarpass := false
	for _, e := range result.Effects {
		if e.Type == "starpass" {
			hasStarpass = true
		}
	}
	if !hasStarpass {
		t.Errorf("imp self-kill should produce starpass effect")
	}
}

func TestUndertakerWithExecution(t *testing.T) {
	ctx := buildTestContext()
	ctx.NightNumber = 2
	ctx.Players["undertaker"] = &PlayerState{UserID: "undertaker", SeatNumber: 8, TrueRole: "undertaker", Team: TeamGood, IsAlive: true}
	ctx.ExecutedToday = "poison" // poisoner was executed today
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:       "undertaker",
		RoleID:       "undertaker",
		NightNumber:  2,
		IsFirstNight: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success")
	}
	if result.Information == nil {
		t.Fatalf("expected information")
	}
	content, ok := result.Information.Content.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map content")
	}
	if content["role"] != "poisoner" {
		t.Errorf("expected undertaker to learn role=poisoner, got %v", content["role"])
	}
}

func TestUndertakerNoExecution(t *testing.T) {
	ctx := buildTestContext()
	ctx.NightNumber = 2
	ctx.Players["undertaker"] = &PlayerState{UserID: "undertaker", SeatNumber: 8, TrueRole: "undertaker", Team: TeamGood, IsAlive: true}
	// No execution today
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:       "undertaker",
		RoleID:       "undertaker",
		NightNumber:  2,
		IsFirstNight: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success")
	}
	if result.Information == nil {
		t.Fatalf("expected information")
	}
	content, ok := result.Information.Content.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map content")
	}
	if _, hasNoExec := content["no_execution"]; !hasNoExec {
		t.Errorf("expected no_execution flag when nobody was executed")
	}
}

func TestPoisonedUndertakerGetsFakeRole(t *testing.T) {
	ctx := buildTestContext()
	ctx.NightNumber = 2
	ctx.Players["undertaker"] = &PlayerState{UserID: "undertaker", SeatNumber: 8, TrueRole: "undertaker", Team: TeamGood, IsAlive: true}
	ctx.PoisonedIDs["undertaker"] = true
	ctx.ExecutedToday = "poison"
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:       "undertaker",
		RoleID:       "undertaker",
		NightNumber:  2,
		IsFirstNight: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success")
	}
	if !result.IsPoisoned {
		t.Errorf("expected IsPoisoned flag")
	}
	if result.Information == nil || !result.Information.IsFalse {
		t.Errorf("expected false information from poisoned undertaker")
	}
}

func TestFortuneTellerRedHerring(t *testing.T) {
	ctx := buildTestContext()
	ctx.RedHerringID = "washer" // washerwoman is the red herring
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:    "fort",
		RoleID:    "fortune_teller",
		TargetIDs: []string{"washer", "empath"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Information == nil {
		t.Fatalf("expected information")
	}
	// Red herring should trigger "has demon"
	content := result.Information.Content.(map[string]interface{})
	if content["has_demon"] != true {
		t.Errorf("expected red herring to register as demon, got %v", content["has_demon"])
	}
}

func TestFortuneTellerNoRedHerring(t *testing.T) {
	ctx := buildTestContext()
	ctx.RedHerringID = "monk" // monk is the red herring, not in this query
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:    "fort",
		RoleID:    "fortune_teller",
		TargetIDs: []string{"washer", "empath"},
	})
	if err != nil {
		t.Fatal(err)
	}
	content := result.Information.Content.(map[string]interface{})
	if content["has_demon"] != false {
		t.Errorf("expected no demon detected (red herring not in query)")
	}
}

func TestFortuneTellerFindsRealDemon(t *testing.T) {
	ctx := buildTestContext()
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:    "fort",
		RoleID:    "fortune_teller",
		TargetIDs: []string{"imp", "washer"},
	})
	if err != nil {
		t.Fatal(err)
	}
	content := result.Information.Content.(map[string]interface{})
	if content["has_demon"] != true {
		t.Errorf("expected demon detected when imp is target")
	}
}

func TestEmpathCountsEvilNeighbors(t *testing.T) {
	ctx := buildTestContext()
	// SeatOrder: washer, empath, fort, monk, butler, poison, imp
	// empath's neighbors: washer (good) and fort (good) -> 0 evil
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID: "empath",
		RoleID: "empath",
	})
	if err != nil {
		t.Fatal(err)
	}
	content := result.Information.Content.(map[string]interface{})
	if content["evil_neighbors"] != 0 {
		t.Errorf("expected 0 evil neighbors for empath between washer and fort, got %v", content["evil_neighbors"])
	}
}

func TestEmpathWithEvilNeighbor(t *testing.T) {
	ctx := buildTestContext()
	// Move empath next to poison: washer, poison, empath, fort, monk, butler, imp
	ctx.SeatOrder = []string{"washer", "poison", "empath", "fort", "monk", "butler", "imp"}
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID: "empath",
		RoleID: "empath",
	})
	if err != nil {
		t.Fatal(err)
	}
	content := result.Information.Content.(map[string]interface{})
	if content["evil_neighbors"] != 1 {
		t.Errorf("expected 1 evil neighbor, got %v", content["evil_neighbors"])
	}
}

func TestRecluseRegistersAsEvilForEmpath(t *testing.T) {
	ctx := buildTestContext()
	ctx.Players["recluse"] = &PlayerState{UserID: "recluse", SeatNumber: 8, TrueRole: "recluse", Team: TeamGood, IsAlive: true}
	// SeatOrder: empath between recluse and fort
	ctx.SeatOrder = []string{"washer", "recluse", "empath", "fort", "monk", "butler", "poison", "imp"}
	ctx.RecluseRegisterEvil = true // Storyteller decided recluse registers as evil
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID: "empath",
		RoleID: "empath",
	})
	if err != nil {
		t.Fatal(err)
	}
	content := result.Information.Content.(map[string]interface{})
	if content["evil_neighbors"] != 1 {
		t.Errorf("expected recluse to register as evil, giving 1 evil neighbor, got %v", content["evil_neighbors"])
	}
}

func TestRecluseNotRegisteringAsEvil(t *testing.T) {
	ctx := buildTestContext()
	ctx.Players["recluse"] = &PlayerState{UserID: "recluse", SeatNumber: 8, TrueRole: "recluse", Team: TeamGood, IsAlive: true}
	ctx.SeatOrder = []string{"washer", "recluse", "empath", "fort", "monk", "butler", "poison", "imp"}
	ctx.RecluseRegisterEvil = false // Recluse doesn't register as evil this night
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID: "empath",
		RoleID: "empath",
	})
	if err != nil {
		t.Fatal(err)
	}
	content := result.Information.Content.(map[string]interface{})
	if content["evil_neighbors"] != 0 {
		t.Errorf("expected 0 evil neighbors when recluse not registering, got %v", content["evil_neighbors"])
	}
}

func TestRecluseRegistersAsDemonForFortuneTeller(t *testing.T) {
	ctx := buildTestContext()
	ctx.Players["recluse"] = &PlayerState{UserID: "recluse", SeatNumber: 8, TrueRole: "recluse", Team: TeamGood, IsAlive: true}
	ctx.RecluseRegisterEvil = true
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:    "fort",
		RoleID:    "fortune_teller",
		TargetIDs: []string{"recluse", "washer"},
	})
	if err != nil {
		t.Fatal(err)
	}
	content := result.Information.Content.(map[string]interface{})
	if content["has_demon"] != true {
		t.Errorf("expected recluse to register as demon for fortune teller")
	}
}

func TestChefCountsEvilPairs(t *testing.T) {
	ctx := buildTestContext()
	// SeatOrder: ..., poison(evil), imp(evil) are adjacent -> 1 pair
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:       "washer",
		RoleID:       "chef",
		IsFirstNight: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	content := result.Information.Content.(map[string]interface{})
	// poison(seat 6) and imp(seat 7) are adjacent
	if content["pairs"] != 1 {
		t.Errorf("expected 1 evil pair (poison+imp adjacent), got %v", content["pairs"])
	}
}

func TestChefWithRecluseEvil(t *testing.T) {
	ctx := buildTestContext()
	ctx.Players["recluse"] = &PlayerState{UserID: "recluse", SeatNumber: 8, TrueRole: "recluse", Team: TeamGood, IsAlive: true}
	// Put recluse next to poison: ..., recluse, poison, imp
	ctx.SeatOrder = []string{"washer", "empath", "fort", "monk", "butler", "recluse", "poison", "imp"}
	ctx.RecluseRegisterEvil = true
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:       "washer",
		RoleID:       "chef",
		IsFirstNight: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	content := result.Information.Content.(map[string]interface{})
	// recluse(evil)+poison(evil) is a pair, poison(evil)+imp(evil) is a pair = 2 pairs
	if content["pairs"] != 2 {
		t.Errorf("expected 2 evil pairs with recluse registering evil, got %v", content["pairs"])
	}
}

func TestMonkProtectsPlayer(t *testing.T) {
	ctx := buildTestContext()
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:       "monk",
		RoleID:       "monk",
		TargetIDs:    []string{"washer"},
		NightNumber:  2,
		IsFirstNight: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success")
	}
	hasProtect := false
	for _, e := range result.Effects {
		if e.Type == "protect" && e.TargetID == "washer" {
			hasProtect = true
		}
	}
	if !hasProtect {
		t.Errorf("expected protect effect on washer")
	}
}

func TestPoisonerPoisonsPlayer(t *testing.T) {
	ctx := buildTestContext()
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:    "poison",
		RoleID:    "poisoner",
		TargetIDs: []string{"empath"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success")
	}
	hasPoison := false
	for _, e := range result.Effects {
		if e.Type == "poison" && e.TargetID == "empath" {
			hasPoison = true
		}
	}
	if !hasPoison {
		t.Errorf("expected poison effect on empath")
	}
}

func TestButlerChoosesMaster(t *testing.T) {
	ctx := buildTestContext()
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:    "butler",
		RoleID:    "butler",
		TargetIDs: []string{"monk"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success")
	}
	hasMaster := false
	for _, e := range result.Effects {
		if e.Type == "butler_master" && e.TargetID == "monk" {
			hasMaster = true
		}
	}
	if !hasMaster {
		t.Errorf("expected butler_master effect")
	}
}

func TestButlerCannotChooseSelf(t *testing.T) {
	ctx := buildTestContext()
	na := NewNightAgent(ctx)

	result, err := na.ResolveAbility(AbilityRequest{
		UserID:    "butler",
		RoleID:    "butler",
		TargetIDs: []string{"butler"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Errorf("butler should not be able to choose self as master")
	}
}
