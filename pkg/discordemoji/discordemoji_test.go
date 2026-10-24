package discordemoji

import (
	"maps"
	"slices"
	"testing"
)

func TestAll(t *testing.T) {
	var names []string
	for name := range All() {
		names = append(names, name)
	}

	// Discord's emoji picker starts with grinning, also named grinning_face.
	if !slices.Equal(names[:2], []string{"grinning", "grinning_face"}) {
		t.Errorf("first names = %v, want [grinning grinning_face]", names[:2])
	}

	if len(names) != len(entries) {
		t.Errorf("len = %d; want %d", len(names), len(entries))
	}

	emojis := maps.Collect(All())
	for name, want := range map[string]string{
		"thumbsup":       "👍",
		"thumbsup_tone3": "👍🏽",
		"piñata":         "🪅",
		"pinata":         "🪅",
		"copyright":      "©",
	} {
		if got := emojis[name]; got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}
