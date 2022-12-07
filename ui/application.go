package ui

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/ayntgl/discordo/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func init() {
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeft = 0
	tview.Borders.TopRight = 0
	tview.Borders.BottomLeft = 0
	tview.Borders.BottomRight = 0
	tview.Borders.Horizontal = 0
	tview.Borders.Vertical = 0

	api.UserAgent = fmt.Sprintf("%s/%s %s/%s", config.Name, "0.1", "arikawa", "v3")
	gateway.DefaultIdentity = gateway.IdentifyProperties{
		OS:      runtime.GOOS,
		Browser: config.Name,
		Device:  "",
	}
}

// Application is responsible for initialization and management of the application, widgets, configuration, and state.
type Application struct {
	*tview.Application

	view   *View
	config *config.Config
	state  *state.State
}

func NewApplication(cfg *config.Config) *Application {
	app := &Application{
		Application: tview.NewApplication(),
		config:      cfg,
	}

	app.EnableMouse(app.config.Mouse)
	app.SetBeforeDrawFunc(app.onBeforeDraw)

	// The styles must be assigned before initializing a new view.
	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(cfg.Theme.Background)
	tview.Styles.BorderColor = tcell.GetColor(cfg.Theme.Border)
	tview.Styles.TitleColor = tcell.GetColor(cfg.Theme.Title)

	app.view = newView(app)

	return app
}

func (app *Application) Run(token string) {
	if token != "" {
		if err := app.Connect(token); err != nil {
			log.Fatal(err)
		}

		app.SetRoot(app.view, true)
		app.SetFocus(app.view.GuildsView)
	} else {
		loginView := newLoginView(app)
		app.SetRoot(loginView, true)
	}

	if err := app.Application.Run(); err != nil {
		log.Fatal(err)
	}
}

func (app *Application) Connect(token string) error {
	app.state = state.New(token)
	app.state.AddHandler(app.onReady)
	app.state.AddHandler(app.onGuildCreate)
	app.state.AddHandler(app.onGuildDelete)
	app.state.AddHandler(app.onMessageCreate)

	return app.state.Open(context.Background())
}

func (app *Application) onBeforeDraw(screen tcell.Screen) bool {
	if app.config.Theme.Background == "default" {
		screen.Clear()
	}

	return false
}

func (c *Application) onReady(r *gateway.ReadyEvent) {
	root := c.view.GuildsView.GetRoot()
	for _, gf := range r.UserSettings.GuildFolders {
		if gf.ID == 0 {
			for _, gID := range gf.GuildIDs {
				g, err := c.state.Cabinet.Guild(gID)
				if err != nil {
					log.Println(err)
					continue
				}

				guildNode := tview.NewTreeNode(g.Name)
				guildNode.SetReference(g.ID)
				root.AddChild(guildNode)
			}
		} else {
			var b strings.Builder

			if gf.Color != discord.NullColor {
				b.WriteByte('[')
				b.WriteString(gf.Color.String())
				b.WriteByte(']')
			} else {
				b.WriteString("[#ED4245]")
			}

			if gf.Name != "" {
				b.WriteString(gf.Name)
			} else {
				b.WriteString("Folder")
			}

			b.WriteString("[-]")

			folderNode := tview.NewTreeNode(b.String())
			root.AddChild(folderNode)

			for _, gID := range gf.GuildIDs {
				g, err := c.state.Cabinet.Guild(gID)
				if err != nil {
					log.Println(err)
					continue
				}

				guildNode := tview.NewTreeNode(g.Name)
				guildNode.SetReference(g.ID)
				folderNode.AddChild(guildNode)
			}
		}

	}

	c.view.GuildsView.SetCurrentNode(root)
	c.SetFocus(c.view.GuildsView)
}

func (c *Application) onGuildCreate(g *gateway.GuildCreateEvent) {
	guildNode := tview.NewTreeNode(g.Name)
	guildNode.SetReference(g.ID)

	rootNode := c.view.GuildsView.GetRoot()
	rootNode.AddChild(guildNode)

	c.view.GuildsView.SetCurrentNode(rootNode)
	c.SetFocus(c.view.GuildsView)
	c.Draw()
}

func (c *Application) onGuildDelete(g *gateway.GuildDeleteEvent) {
	rootNode := c.view.GuildsView.GetRoot()
	var parentNode *tview.TreeNode
	rootNode.Walk(func(node, _ *tview.TreeNode) bool {
		if node.GetReference() == g.ID {
			parentNode = node
			return false
		}

		return true
	})

	if parentNode != nil {
		rootNode.RemoveChild(parentNode)
	}

	c.Draw()
}

func (c *Application) onMessageCreate(m *gateway.MessageCreateEvent) {
	if c.view.ChannelsView.selected != nil && m.ChannelID == c.view.ChannelsView.selected.ID {
		_, err := c.view.MessagesView.Write(buildMessage(c, m.Message))
		if err != nil {
			return
		}

		if len(c.view.MessagesView.GetHighlights()) == 0 {
			c.view.MessagesView.ScrollToEnd()
		}
	}
}
