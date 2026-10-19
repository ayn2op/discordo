package command

import (
	"strings"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
)

func TestBuiltinsSearch(t *testing.T) {
	// Query "shrug"
	results := Search("shrug", 0, discord.PermissionSendMessages, nil, 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'shrug', got none")
	}
	if results[0].Name != "shrug" {
		t.Errorf("expected top result 'shrug', got %q", results[0].Name)
	}

	// Query "flip"
	flipResults := Search("flip", 0, discord.PermissionSendMessages, nil, 10)
	if len(flipResults) < 2 {
		t.Fatalf("expected at least 2 flip results, got %d", len(flipResults))
	}
}

func TestPermissionFiltering(t *testing.T) {
	perm := discord.Permissions(discord.PermissionManageMessages)
	botCmds := []Item{
		{
			Name:         "purge",
			Description:  "Delete messages",
			Source:       "ModBot",
			RequiredPerm: perm,
		},
	}

	// Without manage messages permission
	noPerm := Search("purge", 0, discord.PermissionSendMessages, botCmds, 10)
	if len(noPerm) != 0 {
		t.Errorf("expected purge to be filtered out without permission, got %v", noPerm)
	}

	// With manage messages permission
	withPerm := Search("purge", 0, discord.PermissionSendMessages|discord.PermissionManageMessages, botCmds, 10)
	if len(withPerm) != 1 {
		t.Fatalf("expected purge to be visible with permission, got %v", withPerm)
	}

	// With administrator permission
	admin := Search("purge", 0, discord.PermissionAdministrator, botCmds, 10)
	if len(admin) != 1 {
		t.Fatalf("expected purge to be visible with admin permission, got %v", admin)
	}
}

func TestExpandBuiltin(t *testing.T) {
	tests := []struct {
		input    string
		want     string
		expanded bool
	}{
		{"/shrug", `¯\_(ツ)_/¯`, true},
		{"/shrug hello", `hello ¯\_(ツ)_/¯`, true},
		{"/tableflip", `(╯°□°)╯︵ ┻━┻`, true},
		{"/tableflip angry", `angry (╯°□°)╯︵ ┻━┻`, true},
		{"/unflip", `┬─┬ノ( º _ ºノ)`, true},
		{"/me dancing", `*dancing*`, true},
		{"/spoiler secret text", `||secret text||`, true},
		{"normal message", `normal message`, false},
		{"/unknown command", `/unknown command`, false},
	}

	for _, tt := range tests {
		got, ok := ExpandBuiltin(tt.input)
		if ok != tt.expanded {
			t.Errorf("ExpandBuiltin(%q) ok = %v, want %v", tt.input, ok, tt.expanded)
		}
		if ok && strings.TrimSpace(got) != strings.TrimSpace(tt.want) {
			t.Errorf("ExpandBuiltin(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
