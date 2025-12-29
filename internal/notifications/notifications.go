package notifications

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
)

func Notify(state *ningen.State, message *gateway.MessageCreateEvent, cfg *config.Config) error {
	if !cfg.Notifications.Enabled || cfg.Status == discord.DoNotDisturbStatus {
		return nil
	}

	if state.ChannelIsMuted(message.ChannelID, ningen.UnreadOpts{IncludeMutedCategories: true}) {
		return nil
	}

	mentions := state.MessageMentions(&message.Message)
	if mentions == 0 {
		return nil
	}

	// Handle sent files
	content := message.Content
	if message.Content == "" && len(message.Attachments) > 0 {
		content = "Uploaded " + message.Attachments[0].Filename
	}

	if content == "" {
		return nil
	}

	title := message.Author.DisplayOrUsername()

	channel, err := state.Cabinet.Channel(message.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to get channel from state: %w", err)
	}

	if channel.GuildID.IsValid() {
		guild, err := state.Cabinet.Guild(channel.GuildID)
		if err != nil {
			return fmt.Errorf("failed to get guild from state: %w", err)
		}

		if member := message.Member; member != nil && member.Nick != "" {
			title = member.Nick
		}

		title += " (#" + channel.Name + ", " + guild.Name + ")"
	}

	hash := message.Author.Avatar
	if hash == "" {
		hash = "default"
	}

	imagePath, err := getCachedProfileImage(hash, message.Author.AvatarURLWithType(discord.PNGImage))
	if err != nil {
		slog.Info("failed to get profile image from cache for notification", "err", err, "hash", hash)
	}

	isChannelDM := channel.Type == discord.DirectMessage || channel.Type == discord.GroupDM
	shouldChime := cfg.Notifications.Sound.Enabled && (!cfg.Notifications.Sound.OnlyOnPing || (isChannelDM || mentions.Has(ningen.MessageMentions|ningen.MessageNotifies)))
	if err := sendDesktopNotification(title, content, imagePath, shouldChime, cfg.Notifications.Duration); err != nil {
		return err
	}

	return nil
}

func getCachedProfileImage(avatarHash discord.Hash, url string) (string, error) {
	path := filepath.Join(consts.CacheDir(), "avatars")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", err
	}

	path = filepath.Join(path, avatarHash+".png")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", err
	}

	return path, nil
}
