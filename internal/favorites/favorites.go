// Package favorites keeps the channels bookmarked in the favorite slots and the file they persist in.
//
// A bookmark is the ids of a channel and of the guild it is in, and nothing else, so that a channel
// renamed or moved is still the one the slot opens:
//
//	slot 0 ─┬─ guild id   (absent for a direct message)
//	        └─ channel id
package favorites

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/consts"
)

const fileName = "favorites.toml"

// header tells whoever opens the file what wrote it, as discordo overwrites it whenever a bookmark changes.
const header = "# Written by discordo: the channel bookmarked in each favorite slot, in slot order.\n\n"

// Channel is a bookmarked channel. The zero value is an empty slot.
type Channel struct {
	GuildID   discord.GuildID
	ChannelID discord.ChannelID
}

// Store is the channel bookmarked in each slot, indexed by slot, and the file they are read from and written to.
type Store struct {
	path     string
	channels []Channel
}

// DefaultPath returns the path of the favorites file, next to the configuration file.
func DefaultPath() string {
	return filepath.Join(consts.ConfigDir(), fileName)
}

// Open returns the favorites written in path. A file that cannot be read leaves every slot empty,
// as bookmarks are a convenience that must not keep discordo from starting.
func Open(path string) Store {
	store := Store{path: path}

	var decoded file
	_, err := toml.DecodeFile(path, &decoded)
	switch {
	case os.IsNotExist(err):
		slog.Info("favorites file does not exist, starting with no favorites", "path", path, "err", err)
	case err != nil:
		slog.Error("failed to decode favorites, starting with no favorites", "path", path, "err", err)
	default:
		store.channels = decoded.channels()
	}

	return store
}

// Channel returns the channel bookmarked in slot. It is invalid when the slot is empty.
func (s Store) Channel(slot int) discord.ChannelID {
	if slot < 0 || slot >= len(s.channels) {
		return discord.NullChannelID
	}

	return s.channels[slot].ChannelID
}

// Channels returns the channel bookmarked in each slot, indexed by slot.
func (s Store) Channels() []Channel {
	return slices.Clone(s.channels)
}

// Equal reports whether s and other bookmark the same channels in the same slots.
func (s Store) Equal(other Store) bool {
	return slices.Equal(s.channels, other.channels)
}

// With returns s with channel bookmarked in slot, leaving s as it is. The zero channel empties the slot.
func (s Store) With(slot int, channel Channel) Store {
	if slot < 0 {
		return s
	}

	s.channels = slices.Clone(s.channels)
	// The slots up to the one bookmarked stay empty.
	for len(s.channels) <= slot {
		s.channels = append(s.channels, Channel{})
	}

	s.channels[slot] = channel
	return s
}

// Save writes the favorites, creating the directory they live in if it is not there yet.
func (s Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create the favorites dir: %w", err)
	}

	out, err := os.Create(s.path)
	if err != nil {
		return fmt.Errorf("failed to create the favorites file: %w", err)
	}
	defer out.Close() //nolint:errcheck

	if _, err := out.WriteString(header); err != nil {
		return fmt.Errorf("failed to write the favorites header: %w", err)
	}

	if err := toml.NewEncoder(out).Encode(newFile(s.channels)); err != nil {
		return fmt.Errorf("failed to encode the favorites: %w", err)
	}

	return nil
}

// file is how the favorites are written: a slot per entry, in order, each holding the ids of its channel.
// Ids are strings, the way Discord writes them, and an entry without a channel id is an empty slot.
type file struct {
	Slots []slot `toml:"slots"`
}

type slot struct {
	GuildID   string `toml:"guild_id,omitempty"`
	ChannelID string `toml:"channel_id,omitempty"`
}

func newFile(channels []Channel) file {
	slots := make([]slot, 0, len(channels))
	for _, channel := range channels {
		slots = append(slots, slot{GuildID: channel.GuildID.String(), ChannelID: channel.ChannelID.String()})
	}

	return file{Slots: slots}
}

// channels returns the channel each slot of f holds, leaving the slots that hold no channel empty.
func (f file) channels() []Channel {
	channels := make([]Channel, 0, len(f.Slots))
	for _, slot := range f.Slots {
		channels = append(channels, Channel{GuildID: discord.GuildID(parseID(slot.GuildID)), ChannelID: discord.ChannelID(parseID(slot.ChannelID))})
	}

	return channels
}

// parseID returns the snowflake id holds, or an invalid one if it holds none.
func parseID(id string) discord.Snowflake {
	if id == "" {
		return discord.NullSnowflake
	}

	parsed, err := discord.ParseSnowflake(id)
	if err != nil {
		slog.Error("failed to parse a favorite id, leaving its slot empty", "id", id, "err", err)
		return discord.NullSnowflake
	}

	return parsed
}
