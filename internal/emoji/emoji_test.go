package emoji

import (
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
)

func TestSearchUnicode(t *testing.T) {
	results := Search("smile", nil, 5)
	if len(results) == 0 {
		t.Fatal("expected results for 'smile', got none")
	}

	found := false
	for _, item := range results {
		if item.Name == "smile" && item.InsertText == "😄" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected smile -> 😄 in results, got %+v", results)
	}
}

func TestSearchCustomEmojis(t *testing.T) {
	custom := []discord.Emoji{
		{
			ID:   123456789,
			Name: "custom_cat",
		},
		{
			ID:       987654321,
			Name:     "pepe_dance",
			Animated: true,
		},
	}

	results := Search("cat", custom, 5)
	if len(results) == 0 {
		t.Fatal("expected results for 'cat', got none")
	}

	found := false
	for _, item := range results {
		if item.Name == "custom_cat" {
			found = true
			if item.InsertText != "<:custom_cat:123456789>" {
				t.Errorf("expected custom format, got %s", item.InsertText)
			}
			if !item.Custom {
				t.Errorf("expected Custom=true")
			}
			break
		}
	}
	if !found {
		t.Errorf("expected custom_cat in results, got %+v", results)
	}

	// Test animated emoji
	animResults := Search("dance", custom, 5)
	if len(animResults) == 0 {
		t.Fatal("expected results for 'dance', got none")
	}
	if animResults[0].InsertText != "<a:pepe_dance:987654321>" {
		t.Errorf("expected animated format, got %s", animResults[0].InsertText)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	custom := []discord.Emoji{
		{
			ID:   111,
			Name: "alpha",
		},
	}
	results := Search("", custom, 10)
	if len(results) == 0 {
		t.Fatal("expected results for empty query, got none")
	}
	// Custom emojis should be listed first
	if results[0].Name != "alpha" {
		t.Errorf("expected custom emoji first for empty query, got %s", results[0].Name)
	}
}
