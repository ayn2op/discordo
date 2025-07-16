package notifications

import (
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

func HandleIncomingMessage(state *ningen.State, message *gateway.MessageCreateEvent, cfg *config.Config) error {
	// Only display notification if enabled and unmuted
	if !cfg.Notifications.Enabled || state.MessageMentions(&message.Message) == 0 || cfg.Identify.Status == discord.DoNotDisturbStatus {
		return nil
	}

	content := message.Content
	// If the message has multiple attachments, the filename of the first attachment is used.
	if content == "" && len(message.Attachments) > 0 {
		content = "Uploaded " + message.Attachments[0].Filename
	}

	if content == "" {
		return nil
	}

	// (discord.Message).GuildID is included in the gateway.MessageCreateEvent struct.
	title := message.Author.DisplayOrUsername()
	if guildID := message.GuildID; guildID.IsValid() {
		guild, err := state.Cabinet.Guild(guildID)
		if err != nil {
			return err
		}

		channel, err := state.Cabinet.Channel(message.ChannelID)
		if err != nil {
			return err
		}

		member, err := state.Cabinet.Member(guildID, message.Author.ID)
		if err != nil {
			return err
		}

		if member.Nick != "" {
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
		slog.Error("Failed to retrieve avatar image for notification", "err", err)
	}

	shouldChime := cfg.Notifications.Sound.Enabled && (!cfg.Notifications.Sound.OnlyOnPing || (!message.GuildID.IsValid() || state.MessageMentions(&message.Message) == 3))
	if err := sendDesktopNotification(title, content, imagePath, shouldChime, cfg.Notifications.Duration); err != nil {
		return err
	}

	return nil
}

func getCachedProfileImage(avatarHash discord.Hash, url string) (string, error) {
	path, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	path = filepath.Join(path, consts.Name, "assets")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", err
	}

	path = filepath.Join(path, avatarHash+".png")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	image, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer image.Close()

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if _, err := io.Copy(image, resp.Body); err != nil {
		return "", err
	}

	return path, nil
}
