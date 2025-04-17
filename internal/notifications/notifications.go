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

func HandleIncomingMessage(s ningen.State, m *gateway.MessageCreateEvent, cfg *config.Config) error {
	// Only display notification if enabled and unmuted
	if !cfg.Notifications.Enabled || s.MessageMentions(&m.Message) == 0 || cfg.Identify.Status == discord.DoNotDisturbStatus {
		return nil
	}

	ch, err := s.Cabinet.Channel(m.ChannelID)
	if err != nil {
		return err
	}

	isChannelDM := ch.Type == discord.DirectMessage || ch.Type == discord.GroupDM
	guild := (*discord.Guild)(nil)
	if !isChannelDM {
		guild, err = s.Cabinet.Guild(ch.GuildID)
		if err != nil {
			return err
		}
	}

	// Render message
	src := []byte(m.Content)
	ast := discordmd.ParseWithMessage(src, *s.Cabinet, &m.Message, false)
	buff := strings.Builder{}
	if err := PlainTextRenderer.Render(&buff, src, ast); err != nil {
		return err
	}

	// Handle sent files
	notifContent := buff.String()
	if m.Message.Content == "" && len(m.Message.Attachments) > 0 {
		notifContent = "Uploaded " + m.Message.Attachments[0].Filename
	}

	if m.Author.DisplayOrTag() == "" || notifContent == "" {
		return nil
	}

	notifTitle := m.Author.DisplayOrTag()
	if guild != nil {
		member, _ := s.Member(ch.GuildID, m.Author.ID)
		if member.Nick != "" {
			notifTitle = member.Nick
		}

		notifTitle = notifTitle + " (#" + ch.Name + ", " + guild.Name + ")"
	}

	hash := m.Author.Avatar
	if hash == "" {
		hash = "default"
	}
	imagePath, err := getCachedProfileImage(hash, m.Author.AvatarURLWithType(discord.PNGImage))
	if err != nil {
		slog.Error("Failed to retrieve avatar image for notification", "err", err)
	}

	shouldChime := cfg.Notifications.Sound.Enabled && (!cfg.Notifications.Sound.OnlyOnPing || (isChannelDM || s.MessageMentions(&m.Message) == 3))
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

	path = filepath.Join(path, consts.Name)
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
