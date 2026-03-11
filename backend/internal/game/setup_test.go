package game

import "testing"

func TestGenerateBluffsExcludesDrunk(t *testing.T) {
	inPlay := []Role{
		{ID: "imp", Type: RoleDemon},
		{ID: "washerwoman", Type: RoleTownsfolk},
		{ID: "poisoner", Type: RoleMinion},
	}

	bluffs := generateBluffs(inPlay, GetRolesByType(RoleTownsfolk), GetRolesByType(RoleOutsider))

	for _, bluff := range bluffs {
		if bluff == "drunk" {
			t.Fatal("expected drunk to be excluded from demon bluff roles")
		}
	}
}

func TestGenerateBluffsStillReturnsAvailableOutsiders(t *testing.T) {
	inPlay := []Role{
		{ID: "imp", Type: RoleDemon},
		{ID: "recluse", Type: RoleOutsider},
		{ID: "saint", Type: RoleOutsider},
	}

	outsiders := []Role{
		{ID: "drunk", Type: RoleOutsider},
		{ID: "recluse", Type: RoleOutsider},
		{ID: "saint", Type: RoleOutsider},
		{ID: "butler", Type: RoleOutsider},
	}

	bluffs := generateBluffs(inPlay, nil, outsiders)

	for _, bluff := range bluffs {
		if bluff == "drunk" {
			t.Fatal("expected drunk to be excluded from demon bluff roles")
		}
	}

	if len(bluffs) != 1 || bluffs[0] != "butler" {
		t.Fatalf("expected butler to remain as the available outsider bluff, got %v", bluffs)
	}
}
