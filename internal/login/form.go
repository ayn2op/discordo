package login

import (
	"errors"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

type DoneFn = func(token string, err error)

type Form struct {
	*tview.Flex
	app           *tview.Application
	errorTextView *tview.TextView

	inputs []*tview.InputField
	active int
	done   DoneFn
}

func NewForm(app *tview.Application, done DoneFn) *Form {
	self := &Form{
		Flex:          tview.NewFlex().SetDirection(tview.FlexRow),
		app:           app,
		errorTextView: tview.NewTextView(),

		done: done,
	}

	emailInput := tview.NewInputField()
	emailInput.
		SetBorder(true).
		SetTitle("Email").
		SetTitleAlign(tview.AlignLeft)

	passwordInput := tview.NewInputField()
	passwordInput.
		SetMaskCharacter('*').
		SetBorder(true).
		SetTitle("Password").
		SetTitleAlign(tview.AlignLeft)

	codeInput := tview.NewInputField()
	codeInput.
		SetMaskCharacter('*').
		SetBorder(true).
		SetTitle("Code (optional)").
		SetTitleAlign(tview.AlignLeft)

	self.inputs = []*tview.InputField{emailInput, passwordInput, codeInput}
	for i, input := range self.inputs {
		var focus bool
		if i == 0 {
			focus = true
		}

		self.AddItem(input, 3, 1, focus)
	}

	self.
		AddItem(self.errorTextView, 0, 1, false).
		SetBorderPadding(0, 0, 1, 1).
		SetInputCapture(self.onInputCapture)
	return self
}

func (self *Form) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyBacktab:
		if self.active == 0 {
			self.active = len(self.inputs) - 1
		} else {
			self.active--
		}

		self.app.SetFocus(self.inputs[self.active])
		return nil
	case tcell.KeyTab:
		if self.active == len(self.inputs)-1 {
			self.active = 0
		} else {
			self.active++
		}

		self.app.SetFocus(self.inputs[self.active])
		return nil

	case tcell.KeyEnter:
		// If the currently active input is not the email input, proceed to login with the provided details.
		if self.active != 0 {
			self.login()
		}
	}

	return event
}

func (self *Form) login() {
	email := self.inputs[0].GetText()
	password := self.inputs[1].GetText()
	if email == "" || password == "" {
		return
	}

	// Create an API client without an authentication token.
	client := api.NewClient("")
	// Spoof the user agent of a web browser.
	client.UserAgent = config.UserAgent

	// Attempt to login using the email and password.
	resp, err := client.Login(email, password)
	if err != nil {
		self.onError(err)
		return
	}

	if resp.Token == "" && resp.MFA {
		code := self.inputs[2].GetText()
		if code == "" {
			self.onError(errors.New("code required"))
			return
		}

		// Attempt to login using the code.
		resp, err = client.TOTP(code, resp.Ticket)
		if err != nil {
			self.onError(err)
			return
		}
	}

	if resp.Token == "" {
		self.onError(errors.New("missing token"))
		return
	}

	go keyring.Set(config.Name, "token", resp.Token)

	if self.done != nil {
		self.done(resp.Token, nil)
	}
}

func (self *Form) onError(err error) {
	self.errorTextView.SetText(err.Error())

	if self.done != nil {
		self.done("", err)
	}
}
