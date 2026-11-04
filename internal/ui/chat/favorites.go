package chat

import (
	"log/slog"
	"slices"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/favorites"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
)

// favorite opens the channel bookmarked in a favorite slot or bookmarks the open channel in one,
// and reports whether msg asks for either.
func (m *Model) favorite(msg tview.KeyMsg) (tview.Cmd, bool) {
	keybinds := m.cfg.Keybinds
	if slot := favoriteSlot(msg, keybinds.Favorites); slot >= 0 {
		return m.openFavorite(slot), true
	}

	if slot := favoriteSlot(msg, keybinds.SetFavorites); slot >= 0 {
		return m.setFavorite(slot), true
	}

	return nil, false
}

// favoriteSlot returns the slot whose keybind msg matches, or -1 when it matches none.
func favoriteSlot(msg tview.KeyMsg, keybinds []config.Keybind) int {
	return slices.IndexFunc(keybinds, func(k config.Keybind) bool {
		return keybind.Matches(msg, k.Keybind)
	})
}

// openFavorite opens the channel bookmarked in slot, doing nothing while the slot is empty.
func (m *Model) openFavorite(slot int) tview.Cmd {
	channelID := m.favorites.Channel(slot)
	if !channelID.IsValid() {
		return nil
	}

	return m.navigateToChannel(channelID)
}

// setFavorite bookmarks the open channel in slot, or empties the slot when that channel is the one bookmarked in it.
func (m *Model) setFavorite(slot int) tview.Cmd {
	channel := m.selectedChannel
	if channel == nil {
		return nil
	}

	bookmark := favorites.Channel{GuildID: channel.GuildID, ChannelID: channel.ID}
	if m.favorites.Channel(slot) == channel.ID {
		bookmark = favorites.Channel{}
	}

	m.favorites = m.favorites.With(slot, bookmark)
	return saveFavorites(m.favorites)
}

// saveFavorites writes the bookmarks. A file it cannot write is only logged, as losing a bookmark must not interrupt the chat.
func saveFavorites(store favorites.Store) tview.Cmd {
	return func() tview.Msg {
		if err := store.Save(); err != nil {
			slog.Error("failed to save the favorites", "err", err)
		}

		return nil
	}
}
