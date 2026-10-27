// Package discordemoji maps Discord's emoji names to Unicode emojis.
package discordemoji

import "iter"

//go:generate go run generator.go

type entry struct{ name, emoji string }

// All returns the names and emojis in the order of Discord's emoji picker, each emoji's names together, its main name first.
func All() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, e := range entries {
			if !yield(e.name, e.emoji) {
				return
			}
		}
	}
}
