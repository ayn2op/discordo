package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewMainFlex(app *App) *tview.Flex {
	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		return onAppInputCapture(app, e)
	})

	app.GuildsList.
		ShowSecondaryText(false).
		SetSelectedFunc(func(guildIdx int, _ string, _ string, _ rune) {
			onGuildsListSelected(app, guildIdx)
		}).
		SetTitle("Guilds").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	app.ChannelsTreeView.
		SetTopLevel(1).
		SetRoot(tview.NewTreeNode("")).
		SetSelectedFunc(func(n *tview.TreeNode) {
			onChannelsTreeViewSelected(app, n)
		}).
		SetTitle("Channels").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	app.MessagesTextView.
		SetRegions(true).
		SetDynamicColors(true).
		SetWordWrap(true).
		SetChangedFunc(func() { app.Draw() }).
		SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
			return onMessagesTextViewInputCapture(app, e)
		}).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	app.MessageInputField.
		SetPlaceholder("Message...").
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetPlaceholderStyle(tcell.StyleDefault.Background(tview.Styles.PrimitiveBackgroundColor)).
		SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
			return onMessageInputFieldInputCapture(app, e)
		}).
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	leftFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(app.GuildsList, 10, 1, false).
		AddItem(app.ChannelsTreeView, 0, 1, false)
	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(app.MessagesTextView, 0, 1, false).
		AddItem(app.MessageInputField, 3, 1, false)
	app.MainFlex.
		AddItem(leftFlex, 0, 1, false).
		AddItem(rightFlex, 0, 4, false)

	return app.MainFlex
}

func NewLoginForm(mfa bool) *tview.Form {
	loginForm := tview.NewForm()
	loginForm.
		SetButtonsAlign(tview.AlignCenter).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	if mfa {
		loginForm.AddPasswordField("MFA Code (optional)", "", 0, 0, nil)
	} else {
		loginForm.
			AddInputField("Email", "", 0, nil, nil).
			AddPasswordField("Password", "", 0, 0, nil)
	}

	return loginForm
}
