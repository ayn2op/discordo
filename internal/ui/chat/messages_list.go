package chat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/discordo/internal/media"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v3"
	"github.com/skratchdot/open-golang/open"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type ImageCacheEntry struct {
	ID     uint32
	Width  int
	Height int
	Loaded bool
	Failed bool
}

const (
	maxImageDownloadSize = 25 * 1024 * 1024
)

var imageHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

type messagesList struct {
	*tview.TextView
	cfg               *config.Config
	chatView          *View
	selectedMessageID discord.MessageID

	renderer *markdown.Renderer

	fetchingMembers struct {
		mu    sync.Mutex
		value bool
		count uint
		done  chan struct{}
	}

	imageCache   map[string]ImageCacheEntry
	imageCacheMu sync.RWMutex
	imageCtx     context.Context
	imageCancel  context.CancelFunc

	imageState struct {
		mu          sync.Mutex
		lastOffset  int
		lastBounds  image.Rectangle
		dirty       atomic.Bool
		scrolling   atomic.Bool
		scrollTimer *time.Timer
		timerGen    uint64
		imagesDrawn bool
	}
}

func newMessagesList(cfg *config.Config, chatView *View) *messagesList {
	ctx, cancel := context.WithCancel(context.Background())
	ml := &messagesList{
		TextView:    tview.NewTextView(),
		cfg:         cfg,
		chatView:    chatView,
		renderer:    markdown.NewRenderer(cfg.Theme.MessagesList),
		imageCache:  make(map[string]ImageCacheEntry),
		imageCtx:    ctx,
		imageCancel: cancel,
	}

	ml.Box = ui.ConfigureBox(ml.Box, &cfg.Theme)
	ml.
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		ScrollToEnd().
		SetHighlightedFunc(ml.onHighlighted).
		SetTitle("Messages").
		SetInputCapture(ml.onInputCapture)
	return ml
}

func (ml *messagesList) reset() {
	ml.selectedMessageID = 0
	ml.clearImageCache()
	ml.
		Clear().
		Highlight().
		SetTitle("")
}

func (ml *messagesList) clearImageCache() {
	if ml.imageCancel != nil {
		ml.imageCancel()
	}
	ml.imageCtx, ml.imageCancel = context.WithCancel(context.Background())

	ml.imageCacheMu.Lock()
	ml.imageCache = make(map[string]ImageCacheEntry)
	ml.imageCacheMu.Unlock()
	ml.imageState.dirty.Store(true)
	media.GlobalImageManager.ClearAll()
}

func (ml *messagesList) Draw(screen tcell.Screen) {
	ml.TextView.Draw(screen)

	if !ml.cfg.ImagePreviews.Enabled {
		return
	}

	proto := media.DetectProtocol()
	if proto == media.ProtoFallback {
		return
	}

	x, y, w, h := ml.GetInnerRect()
	if w <= 0 || h <= 0 {
		return
	}

	currentOffset, _ := ml.GetScrollOffset()
	currentBounds := image.Rect(x, y, x+w, y+h)

	ml.imageState.mu.Lock()
	offsetChanged := currentOffset != ml.imageState.lastOffset
	boundsChanged := currentBounds != ml.imageState.lastBounds
	needsRedraw := ml.imageState.dirty.Load() || offsetChanged || boundsChanged || !ml.imageState.imagesDrawn

	if offsetChanged {
		ml.imageState.scrolling.Store(true)

		if ml.imageState.scrollTimer != nil {
			ml.imageState.scrollTimer.Stop()
		}

		ml.imageState.timerGen++
		gen := ml.imageState.timerGen

		ml.imageState.scrollTimer = time.AfterFunc(150*time.Millisecond, func() {
			ml.imageState.mu.Lock()
			if ml.imageState.timerGen != gen {
				ml.imageState.mu.Unlock()
				return
			}
			ml.imageState.mu.Unlock()

			ml.imageState.scrolling.Store(false)
			ml.imageState.dirty.Store(true)
			ml.chatView.app.QueueUpdateDraw(func() {})
		})
	}

	ml.imageState.lastOffset = currentOffset
	ml.imageState.lastBounds = currentBounds
	ml.imageState.mu.Unlock()

	if ml.imageState.scrolling.Load() && proto == media.ProtoSixel {
		return
	}

	if !needsRedraw {
		return
	}

	ml.imageState.dirty.Store(false)

	media.GlobalImageManager.ClearScreen(screen)
	media.GlobalImageManager.ScanAndDrawWithClip(screen, x, y, w, h, currentBounds)

	ml.imageState.mu.Lock()
	ml.imageState.imagesDrawn = true
	ml.imageState.mu.Unlock()
}

func (ml *messagesList) setTitle(channel discord.Channel) {
	title := ui.ChannelToString(channel)
	if topic := channel.Topic; topic != "" {
		title += " - " + topic
	}

	ml.SetTitle(title)
}

func (ml *messagesList) drawMessages(messages []discord.Message) {
	writer := ml.BatchWriter()
	defer writer.Close()
	for _, m := range slices.Backward(messages) {
		ml.drawMessage(writer, m)
	}
}

func (ml *messagesList) drawMessage(writer io.Writer, message discord.Message) {
	// Region tags are square brackets that contain a region ID in double quotes
	// https://pkg.go.dev/github.com/ayn2op/tview#hdr-Regions_and_Highlights
	fmt.Fprintf(writer, `["%s"]`, message.ID)

	if ml.cfg.HideBlockedUsers {
		isBlocked := ml.chatView.state.UserIsBlocked(message.Author.ID)
		if isBlocked {
			io.WriteString(writer, "[:red:b]Blocked message[:-:-]")
			return
		}
	}

	// reset
	io.WriteString(writer, "[-:-:-]")

	switch message.Type {
	case discord.DefaultMessage:
		if message.Reference != nil && message.Reference.Type == discord.MessageReferenceTypeForward {
			ml.drawForwardedMessage(writer, message)
		} else {
			ml.drawDefaultMessage(writer, message)
		}
	case discord.GuildMemberJoinMessage:
		ml.drawTimestamps(writer, message.Timestamp)
		ml.drawAuthor(writer, message)
		fmt.Fprint(writer, "joined the server.")
	case discord.InlinedReplyMessage:
		ml.drawReplyMessage(writer, message)
	case discord.ChannelPinnedMessage:
		ml.drawPinnedMessage(writer, message)
	default:
		ml.drawTimestamps(writer, message.Timestamp)
		ml.drawAuthor(writer, message)
	}

	// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
	io.WriteString(writer, "[\"\"]\n")
}

func (ml *messagesList) formatTimestamp(ts discord.Timestamp) string {
	return ts.Time().In(time.Local).Format(ml.cfg.Timestamps.Format)
}

func (ml *messagesList) drawTimestamps(w io.Writer, ts discord.Timestamp) {
	fmt.Fprintf(w, "[::d]%s[::D] ", ml.formatTimestamp(ts))
}

func (ml *messagesList) drawAuthor(w io.Writer, message discord.Message) {
	name := message.Author.DisplayOrUsername()
	foreground := tcell.ColorDefault

	// Webhooks do not have nicknames or roles.
	if message.GuildID.IsValid() && !message.WebhookID.IsValid() {
		member, err := ml.chatView.state.Cabinet.Member(message.GuildID, message.Author.ID)
		if err != nil {
			slog.Error("failed to get member from state", "guild_id", message.GuildID, "member_id", message.Author.ID, "err", err)
		} else {
			if member.Nick != "" {
				name = member.Nick
			}

			color, ok := state.MemberColor(member, func(id discord.RoleID) *discord.Role {
				r, _ := ml.chatView.state.Cabinet.Role(message.GuildID, id)
				return r
			})
			if ok {
				foreground = tcell.NewHexColor(int32(color))
			}
		}
	}

	fmt.Fprintf(w, "[%s::b]%s[-::B] ", foreground, name)
}

func (ml *messagesList) drawContent(w io.Writer, message discord.Message) {
	c := []byte(tview.Escape(message.Content))
	if ml.chatView.cfg.Markdown {
		ast := discordmd.ParseWithMessage(c, *ml.chatView.state.Cabinet, &message, false)
		ml.renderer.Render(w, c, ast)
	} else {
		w.Write(c) // write the content as is
	}
}

func (ml *messagesList) drawSnapshotContent(w io.Writer, message discord.MessageSnapshotMessage) {
	c := []byte(tview.Escape(message.Content))
	// discordmd doesn't support MessageSnapshotMessage, so we just use write it as is. todo?
	w.Write(c)
}

func (ml *messagesList) drawDefaultMessage(w io.Writer, message discord.Message) {
	if ml.cfg.Timestamps.Enabled {
		ml.drawTimestamps(w, message.Timestamp)
	}

	ml.drawAuthor(w, message)
	ml.drawContent(w, message)

	if message.EditedTimestamp.IsValid() {
		io.WriteString(w, " [::d](edited)[::D]")
	}

	for _, a := range message.Attachments {
		fmt.Fprintln(w)

		if ml.cfg.ImagePreviews.Enabled && strings.HasPrefix(a.ContentType, "image/") && media.DetectProtocol() != media.ProtoFallback {
			ml.imageCacheMu.RLock()
			entry, ok := ml.imageCache[a.URL]
			ml.imageCacheMu.RUnlock()

			if ok {
				if entry.Failed {
					fg := ml.cfg.Theme.MessagesList.AttachmentStyle.GetForeground()
					bg := ml.cfg.Theme.MessagesList.AttachmentStyle.GetBackground()
					fmt.Fprintf(w, "[%s:%s][Failed to load image: %s][-:-]\n", fg, bg, a.Filename)
				} else if entry.Loaded {
					fmt.Fprintf(w, "[#%06x] [:-]\n", entry.ID)
					for r := 1; r < entry.Height; r++ {
						fmt.Fprintln(w)
					}
				} else {
					fmt.Fprint(w, "[Loading image...]\n")
				}
			} else {
				fmt.Fprint(w, "[Loading image...]\n")
				ml.imageCacheMu.Lock()
				ml.imageCache[a.URL] = ImageCacheEntry{Loaded: false}
				ml.imageCacheMu.Unlock()
				go ml.downloadImage(a.URL)
			}
			continue
		}

		fg := ml.cfg.Theme.MessagesList.AttachmentStyle.GetForeground()
		bg := ml.cfg.Theme.MessagesList.AttachmentStyle.GetBackground()
		if ml.cfg.ShowAttachmentLinks {
			fmt.Fprintf(w, "[%s:%s]%s:\n%s[-:-]", fg, bg, a.Filename, a.URL)
		} else {
			fmt.Fprintf(w, "[%s:%s]%s[-:-]", fg, bg, a.Filename)
		}
	}
}

func (ml *messagesList) downloadImage(url string) {
	ctx := ml.imageCtx

	markFailed := func() {
		if ctx.Err() != nil {
			return
		}
		ml.imageCacheMu.Lock()
		ml.imageCache[url] = ImageCacheEntry{Failed: true}
		ml.imageCacheMu.Unlock()
		ml.chatView.app.QueueUpdateDraw(func() {
			ml.reDraw()
		})
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		slog.Error("failed to create image request", "err", err)
		markFailed()
		return
	}

	resp, err := imageHTTPClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("failed to download image", "err", err)
		markFailed()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("failed to download image", "status", resp.StatusCode, "url", url)
		markFailed()
		return
	}

	if resp.ContentLength > maxImageDownloadSize {
		slog.Warn("image too large, skipping", "url", url, "size", resp.ContentLength, "max", maxImageDownloadSize)
		markFailed()
		return
	}

	limitedReader := io.LimitReader(resp.Body, maxImageDownloadSize+1)
	imageData, err := io.ReadAll(limitedReader)
	if err != nil {
		slog.Error("failed to read image data", "err", err)
		markFailed()
		return
	}

	if len(imageData) > maxImageDownloadSize {
		slog.Warn("image too large, skipping", "url", url, "size", len(imageData), "max", maxImageDownloadSize)
		markFailed()
		return
	}

	imgCfg, _, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		slog.Error("failed to decode image config", "err", err)
		markFailed()
		return
	}

	maxPixels := 100 * 1024 * 1024
	if imgCfg.Width*imgCfg.Height > maxPixels {
		slog.Warn("image too many pixels", "url", url, "pixels", imgCfg.Width*imgCfg.Height, "max", maxPixels)
		markFailed()
		return
	}

	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		slog.Error("failed to decode image", "err", err)
		markFailed()
		return
	}

	cellW, cellH := media.GetCellSize()

	maxW := ml.cfg.ImagePreviews.MaxWidth
	maxH := ml.cfg.ImagePreviews.MaxHeight

	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()

	maxWPixels := maxW * cellW
	maxHPixels := maxH * cellH

	scaleW := float64(maxWPixels) / float64(srcW)
	scaleH := float64(maxHPixels) / float64(srcH)

	scale := min(scaleW, scaleH)
	if scale > 1 {
		scale = 1
	}

	newW := int(float64(srcW) * scale)
	newH := int(float64(srcH) * scale)

	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	cellCols := (newW + cellW - 1) / cellW
	cellRows := (newH + cellH - 1) / cellH

	if cellCols > maxW {
		cellCols = maxW
	}
	if cellRows > maxH {
		cellRows = maxH
	}
	if cellCols < 1 {
		cellCols = 1
	}
	if cellRows < 1 {
		cellRows = 1
	}

	if ctx.Err() != nil {
		return
	}

	id := media.GlobalImageManager.Register(url, cellCols, cellRows)
	media.GlobalImageManager.SetImage(id, dst)

	if ctx.Err() != nil {
		return
	}

	ml.imageCacheMu.Lock()
	ml.imageCache[url] = ImageCacheEntry{
		ID:     id,
		Width:  cellCols,
		Height: cellRows,
		Loaded: true,
	}
	ml.imageCacheMu.Unlock()

	ml.chatView.app.QueueUpdateDraw(func() {
		ml.reDraw()
	})
}

func (ml *messagesList) reDraw() {
	selected := ml.chatView.SelectedChannel()
	if selected == nil {
		return
	}
	messages, err := ml.chatView.state.Cabinet.Messages(selected.ID)
	if err != nil {
		return
	}

	title := ml.GetTitle()
	ml.Clear()
	ml.drawMessages(messages)
	ml.SetTitle(title)
	ml.imageState.dirty.Store(true)

	if ml.selectedMessageID.IsValid() {
		ml.Highlight(ml.selectedMessageID.String())
		ml.ScrollToHighlight()
	}
}

func (ml *messagesList) drawForwardedMessage(w io.Writer, message discord.Message) {
	ml.drawTimestamps(w, message.Timestamp)
	ml.drawAuthor(w, message)
	fmt.Fprintf(w, "[::d]%s [::-]", ml.cfg.Theme.MessagesList.ForwardedIndicator)
	ml.drawSnapshotContent(w, message.MessageSnapshots[0].Message)
	fmt.Fprintf(w, " [::d](%s)[-:-:-] ", ml.formatTimestamp(message.MessageSnapshots[0].Message.Timestamp))
}

func (ml *messagesList) drawReplyMessage(w io.Writer, message discord.Message) {
	// reply
	fmt.Fprintf(w, "[::d]%s ", ml.cfg.Theme.MessagesList.ReplyIndicator)
	if m := message.ReferencedMessage; m != nil {
		m.GuildID = message.GuildID
		ml.drawAuthor(w, *m)
		ml.drawContent(w, *m)
	} else {
		io.WriteString(w, "Original message was deleted")
	}

	io.WriteString(w, "\n")
	// main
	ml.drawDefaultMessage(w, message)
}

func (ml *messagesList) drawPinnedMessage(w io.Writer, message discord.Message) {
	fmt.Fprintf(w, "%s pinned a message", message.Author.DisplayOrUsername())
}

func (ml *messagesList) selectedMessage() (*discord.Message, error) {
	if !ml.selectedMessageID.IsValid() {
		return nil, errors.New("no message is currently selected")
	}

	selected := ml.chatView.SelectedChannel()
	if selected == nil {
		return nil, errors.New("no channel is currently selected")
	}

	m, err := ml.chatView.state.Cabinet.Message(selected.ID, ml.selectedMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve selected message: %w", err)
	}

	return m, nil
}

func (ml *messagesList) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case ml.cfg.Keys.MessagesList.ScrollUp:
		return tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone)
	case ml.cfg.Keys.MessagesList.ScrollDown:
		return tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone)
	case ml.cfg.Keys.MessagesList.ScrollTop:
		return tcell.NewEventKey(tcell.KeyHome, "", tcell.ModNone)
	case ml.cfg.Keys.MessagesList.ScrollBottom:
		return tcell.NewEventKey(tcell.KeyEnd, "", tcell.ModNone)

	case ml.cfg.Keys.MessagesList.Cancel:
		ml.selectedMessageID = 0
		ml.Highlight()

	case ml.cfg.Keys.MessagesList.SelectPrevious, ml.cfg.Keys.MessagesList.SelectNext, ml.cfg.Keys.MessagesList.SelectFirst, ml.cfg.Keys.MessagesList.SelectLast, ml.cfg.Keys.MessagesList.SelectReply:
		ml._select(event.Name())
	case ml.cfg.Keys.MessagesList.YankID:
		ml.yankID()
	case ml.cfg.Keys.MessagesList.YankContent:
		ml.yankContent()
	case ml.cfg.Keys.MessagesList.YankURL:
		ml.yankURL()
	case ml.cfg.Keys.MessagesList.Open:
		ml.open()
	case ml.cfg.Keys.MessagesList.Reply:
		ml.reply(false)
	case ml.cfg.Keys.MessagesList.ReplyMention:
		ml.reply(true)
	case ml.cfg.Keys.MessagesList.Edit:
		ml.edit()
	case ml.cfg.Keys.MessagesList.Delete:
		ml.delete()
	case ml.cfg.Keys.MessagesList.DeleteConfirm:
		ml.confirmDelete()
	}

	return nil
}

func (ml *messagesList) _select(name string) {
	selectedChannel := ml.chatView.SelectedChannel()
	if selectedChannel == nil {
		return
	}

	messages, err := ml.chatView.state.Cabinet.Messages(selectedChannel.ID)
	if err != nil {
		slog.Error("failed to get messages", "err", err, "channel_id", selectedChannel.ID)
		return
	}
	if len(messages) == 0 {
		return
	}

	messageIdx := slices.IndexFunc(messages, func(m discord.Message) bool {
		return m.ID == ml.selectedMessageID
	})
	// Allow "no highlight yet" to fall through and pick the latest message.
	if len(ml.GetHighlights()) != 0 && messageIdx == -1 {
		return
	}

	switch name {
	case ml.cfg.Keys.MessagesList.SelectPrevious:
		// If no message is currently selected, select the latest message.
		if len(ml.GetHighlights()) == 0 {
			ml.selectedMessageID = messages[0].ID
		} else if messageIdx < len(messages)-1 {
			ml.selectedMessageID = messages[messageIdx+1].ID
		} else {
			return
		}
	case ml.cfg.Keys.MessagesList.SelectNext:
		// If no message is currently selected, select the latest message.
		if len(ml.GetHighlights()) == 0 {
			ml.selectedMessageID = messages[0].ID
		} else if messageIdx > 0 {
			ml.selectedMessageID = messages[messageIdx-1].ID
		} else {
			return
		}
	case ml.cfg.Keys.MessagesList.SelectFirst:
		ml.selectedMessageID = messages[len(messages)-1].ID
	case ml.cfg.Keys.MessagesList.SelectLast:
		ml.selectedMessageID = messages[0].ID
	case ml.cfg.Keys.MessagesList.SelectReply:
		if ml.selectedMessageID == 0 {
			return
		}

		if ref := messages[messageIdx].ReferencedMessage; ref != nil {
			refIdx := slices.IndexFunc(messages, func(m discord.Message) bool {
				return m.ID == ref.ID
			})
			if refIdx != -1 {
				ml.selectedMessageID = messages[refIdx].ID
			}
		}
	}

	ml.Highlight(ml.selectedMessageID.String())
	ml.ScrollToHighlight()
}

func (ml *messagesList) onHighlighted(added, removed, remaining []string) {
	if len(added) > 0 {
		id, err := discord.ParseSnowflake(added[0])
		if err != nil {
			slog.Error("failed to parse region id as int to use as message id", "err", err)
			return
		}

		ml.selectedMessageID = discord.MessageID(id)
	}
}

func (ml *messagesList) yankID() {
	msg, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	go clipboard.Write(clipboard.FmtText, []byte(msg.ID.String()))
}

func (ml *messagesList) yankContent() {
	msg, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	go clipboard.Write(clipboard.FmtText, []byte(msg.Content))
}

func (ml *messagesList) yankURL() {
	msg, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	go clipboard.Write(clipboard.FmtText, []byte(msg.URL()))
}

func (ml *messagesList) open() {
	msg, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	var urls []string
	if msg.Content != "" {
		urls = extractURLs(msg.Content)
	}

	if len(urls) == 0 && len(msg.Attachments) == 0 {
		return
	}

	if len(urls)+len(msg.Attachments) == 1 {
		if len(urls) == 1 {
			go ml.openURL(urls[0])
		} else {
			go ml.openAttachment(msg.Attachments[0])
		}
	} else {
		ml.showAttachmentsList(urls, msg.Attachments)
	}
}

func extractURLs(content string) []string {
	src := []byte(content)
	node := parser.NewParser(
		parser.WithBlockParsers(discordmd.BlockParsers()...),
		parser.WithInlineParsers(discordmd.InlineParserWithLink()...),
	).Parse(text.NewReader(src))

	var urls []string
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch n := n.(type) {
			case *ast.AutoLink:
				urls = append(urls, string(n.URL(src)))
			case *ast.Link:
				urls = append(urls, string(n.Destination))
			}
		}

		return ast.WalkContinue, nil
	})
	return urls
}

func (ml *messagesList) showAttachmentsList(urls []string, attachments []discord.Attachment) {
	list := tview.NewList().
		SetWrapAround(true).
		SetHighlightFullLine(true).
		ShowSecondaryText(false).
		SetDoneFunc(func() {
			ml.chatView.RemovePage(attachmentsListPageName).SwitchToPage(flexPageName)
			ml.chatView.app.SetFocus(ml)
		})
	list.
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Name() {
			case ml.cfg.Keys.MessagesList.SelectPrevious:
				return tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone)
			case ml.cfg.Keys.MessagesList.SelectNext:
				return tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone)
			case ml.cfg.Keys.MessagesList.SelectFirst:
				return tcell.NewEventKey(tcell.KeyHome, "", tcell.ModNone)
			case ml.cfg.Keys.MessagesList.SelectLast:
				return tcell.NewEventKey(tcell.KeyEnd, "", tcell.ModNone)
			}

			return event
		})
	list.Box = ui.ConfigureBox(list.Box, &ml.cfg.Theme)

	for i, a := range attachments {
		list.AddItem(a.Filename, "", rune('a'+i), func() {
			go ml.openAttachment(a)
		})
	}

	for i, u := range urls {
		list.AddItem(u, "", rune('1'+i), func() {
			go ml.openURL(u)
		})
	}

	ml.chatView.
		AddAndSwitchToPage(attachmentsListPageName, ui.Centered(list, 0, 0), true).
		ShowPage(flexPageName)
}

func (ml *messagesList) openAttachment(attachment discord.Attachment) {
	resp, err := http.Get(attachment.URL)
	if err != nil {
		slog.Error("failed to fetch the attachment", "err", err, "url", attachment.URL)
		return
	}
	defer resp.Body.Close()

	path := filepath.Join(consts.CacheDir(), "attachments")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		slog.Error("failed to create attachments dir", "err", err, "path", path)
		return
	}

	path = filepath.Join(path, attachment.Filename)
	file, err := os.Create(path)
	if err != nil {
		slog.Error("failed to create attachment file", "err", err, "path", path)
		return
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		slog.Error("failed to copy attachment to file", "err", err)
		return
	}

	if err := open.Start(path); err != nil {
		slog.Error("failed to open attachment file", "err", err, "path", path)
		return
	}
}

func (ml *messagesList) openURL(url string) {
	if err := open.Start(url); err != nil {
		slog.Error("failed to open URL", "err", err, "url", url)
	}
}

func (ml *messagesList) reply(mention bool) {
	msg, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	name := msg.Author.DisplayOrUsername()
	if msg.GuildID.IsValid() {
		member, err := ml.chatView.state.Cabinet.Member(msg.GuildID, msg.Author.ID)
		if err != nil {
			slog.Error("failed to get member from state", "guild_id", msg.GuildID, "member_id", msg.Author.ID, "err", err)
		} else {
			if member.Nick != "" {
				name = member.Nick
			}
		}
	}

	data := ml.chatView.messageInput.sendMessageData
	data.Reference = &discord.MessageReference{MessageID: ml.selectedMessageID}
	data.AllowedMentions = &api.AllowedMentions{RepliedUser: option.False}

	title := "Replying to "
	if mention {
		data.AllowedMentions.RepliedUser = option.True
		title = "[@] " + title
	}

	ml.chatView.messageInput.sendMessageData = data
	ml.chatView.messageInput.SetTitle(title + name)
	ml.chatView.app.SetFocus(ml.chatView.messageInput)
}

func (ml *messagesList) edit() {
	message, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	me, err := ml.chatView.state.Cabinet.Me()
	if err != nil {
		slog.Error("failed to get client user (me)", "err", err)
		return
	}

	if message.Author.ID != me.ID {
		slog.Error("failed to edit message; not the author", "channel_id", message.ChannelID, "message_id", message.ID)
		return
	}

	ml.chatView.messageInput.SetTitle("Editing")
	ml.chatView.messageInput.edit = true
	ml.chatView.messageInput.SetText(message.Content, true)
	ml.chatView.app.SetFocus(ml.chatView.messageInput)
}

func (ml *messagesList) confirmDelete() {
	onChoice := func(choice string) {
		if choice == "Yes" {
			ml.delete()
		}
	}

	ml.chatView.showConfirmModal(
		"Are you sure you want to delete this message?",
		[]string{"Yes", "No"},
		onChoice,
	)
}

func (ml *messagesList) delete() {
	msg, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if msg.GuildID.IsValid() {
		me, err := ml.chatView.state.Cabinet.Me()
		if err != nil {
			slog.Error("failed to get client user (me)", "err", err)
			return
		}

		if msg.Author.ID != me.ID && !ml.chatView.state.HasPermissions(msg.ChannelID, discord.PermissionManageMessages) {
			slog.Error("failed to delete message; missing relevant permissions", "channel_id", msg.ChannelID, "message_id", msg.ID)
			return
		}
	}

	selected := ml.chatView.SelectedChannel()
	if selected == nil {
		return
	}

	if err := ml.chatView.state.DeleteMessage(selected.ID, msg.ID, ""); err != nil {
		slog.Error("failed to delete message", "channel_id", selected.ID, "message_id", msg.ID, "err", err)
		return
	}

	ml.selectedMessageID = 0
	ml.Highlight()

	if err := ml.chatView.state.MessageRemove(selected.ID, msg.ID); err != nil {
		slog.Error("failed to delete message", "channel_id", selected.ID, "message_id", msg.ID, "err", err)
		return
	}

	// No need to redraw messages after deletion, onMessageDelete will do
	// its work after the event returns
}

func (ml *messagesList) requestGuildMembers(guildID discord.GuildID, messages []discord.Message) {
	usersToFetch := make([]discord.UserID, 0, len(messages))
	seen := make(map[discord.UserID]struct{}, len(messages))

	for _, message := range messages {
		// Do not fetch member for a webhook message.
		if message.WebhookID.IsValid() {
			continue
		}

		if member, _ := ml.chatView.state.Cabinet.Member(guildID, message.Author.ID); member == nil {
			userID := message.Author.ID
			if _, ok := seen[userID]; !ok {
				seen[userID] = struct{}{}
				usersToFetch = append(usersToFetch, userID)
			}
		}
	}

	if len(usersToFetch) > 0 {
		err := ml.chatView.state.SendGateway(context.TODO(), &gateway.RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{guildID},
			UserIDs:  usersToFetch,
		})
		if err != nil {
			slog.Error("failed to request guild members", "guild_id", guildID, "err", err)
			return
		}

		ml.setFetchingChunk(true, 0)
		ml.waitForChunkEvent()
	}
}

func (ml *messagesList) setFetchingChunk(value bool, count uint) {
	ml.fetchingMembers.mu.Lock()
	defer ml.fetchingMembers.mu.Unlock()

	if ml.fetchingMembers.value == value {
		return
	}

	ml.fetchingMembers.value = value

	if value {
		ml.fetchingMembers.done = make(chan struct{})
	} else {
		ml.fetchingMembers.count = count
		close(ml.fetchingMembers.done)
	}
}

func (ml *messagesList) waitForChunkEvent() uint {
	ml.fetchingMembers.mu.Lock()
	if !ml.fetchingMembers.value {
		ml.fetchingMembers.mu.Unlock()
		return 0
	}
	ml.fetchingMembers.mu.Unlock()

	<-ml.fetchingMembers.done
	return ml.fetchingMembers.count
}
