package ui

import "github.com/rivo/tview"

// NewMainFlex creates and returns a new main flex.
func NewMainFlex(
	treeV *tview.TreeView,
	textV *tview.TextView,
	i *tview.InputField,
) *tview.Flex {
	rf := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textV, 0, 1, false).
		AddItem(i, 3, 1, false)
	mf := tview.NewFlex().
		AddItem(treeV, 0, 1, false).
		AddItem(rf, 0, 4, false)

	return mf
}

// NewLoginWidget creates and returns a new login widget.
func NewLoginWidget(onLoginFormLoginButtonSelected func(), mfa bool) *tview.Form {
	w := tview.NewForm()
	w.
		AddButton("Login", onLoginFormLoginButtonSelected).
		SetButtonsAlign(tview.AlignCenter).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	if mfa {
		w.AddPasswordField("Code", "", 0, 0, nil)
	} else {
		w.
			AddInputField("Email", "", 0, nil, nil).
			AddPasswordField("Password", "", 0, 0, nil)
	}

	return w
}
