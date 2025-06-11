package notifications

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
	"github.com/diamondburned/ningen/v3/discordmd"
)

func HandleIncomingMessage(state *ningen.State, msg *gateway.MessageCreateEvent, cfg *config.Config) error {
	// Only display notification if enabled and unmuted
	if !cfg.Notifications.Enabled || state.MessageMentions(&msg.Message) == 0 || cfg.Identify.Status == discord.DoNotDisturbStatus {
		return nil
	}

	ch, err := state.Cabinet.Channel(msg.ChannelID)
	if err != nil {
		return err
	}

	isChannelDM := ch.Type == discord.DirectMessage || ch.Type == discord.GroupDM
	guild := (*discord.Guild)(nil)
	if !isChannelDM {
		guild, err = state.Cabinet.Guild(ch.GuildID)
		if err != nil {
			return err
		}
	}

	// Render message
	src := []byte(msg.Content)
	ast := discordmd.ParseWithMessage(src, *state.Cabinet, &msg.Message, false)
	buff := strings.Builder{}
	if err := defaultRenderer.Render(&buff, src, ast); err != nil {
		return err
	}

	// Handle sent files
	notifContent := buff.String()
	if msg.Content == "" && len(msg.Attachments) > 0 {
		notifContent = "Uploaded " + msg.Message.Attachments[0].Filename
	}

	if msg.Author.DisplayOrTag() == "" || notifContent == "" {
		return nil
	}

	notifTitle := msg.Author.DisplayOrTag()
	if guild != nil {
		member, _ := state.Member(ch.GuildID, msg.Author.ID)
		if member.Nick != "" {
			notifTitle = member.Nick
		}

		notifTitle = notifTitle + " (#" + ch.Name + ", " + guild.Name + ")"
	}

	hash := msg.Author.Avatar
	if hash == "" {
		hash = "default"
	}
	imagePath, err := getCachedProfileImage(hash, msg.Author.AvatarURLWithType(discord.PNGImage))
	if err != nil {
		slog.Error("Failed to retrieve avatar image for notification", "err", err)
	}

	shouldChime := cfg.Notifications.Sound.Enabled && (!cfg.Notifications.Sound.OnlyOnPing || (isChannelDM || state.MessageMentions(&msg.Message) == 3))
	if err := sendDesktopNotification(notifTitle, notifContent, imagePath, shouldChime, cfg.Notifications.Duration); err != nil {
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
