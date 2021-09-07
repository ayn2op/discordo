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

func newBaseLoginForm() *tview.Form {
	f := tview.NewForm()
	f.
		SetButtonsAlign(tview.AlignCenter).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return f
}

// NewLoginForm creates and returns a new login form.
func NewLoginForm(onLoginFormLoginButtonSelected func(), mfa bool) *tview.Form {
	f := newBaseLoginForm()
	f.AddButton("Login", onLoginFormLoginButtonSelected)

	if mfa {
		f.AddPasswordField("Code", "", 0, 0, nil)
	} else {
		f.
			AddInputField("Email", "", 0, nil, nil).
			AddPasswordField("Password", "", 0, 0, nil)
	}

	return f
}
