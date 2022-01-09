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
		AddItem("Direct Messages", "", 0, nil).
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
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	app.MessageInputField.
		SetPlaceholder("Message...").
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
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

	return tview.NewFlex().
		AddItem(leftFlex, 0, 1, false).
		AddItem(rightFlex, 0, 4, false)
}

func NewLoginForm(onLoginFormLoginButtonSelected func(), mfa bool) *tview.Form {
	f := tview.NewForm()
	f.
		AddButton("Login", onLoginFormLoginButtonSelected).
		SetButtonsAlign(tview.AlignCenter).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	if mfa {
		f.AddPasswordField("Code", "", 0, 0, nil)
	} else {
		f.
			AddInputField("Email", "", 0, nil, nil).
			AddPasswordField("Password", "", 0, 0, nil)
	}

	return f
}
