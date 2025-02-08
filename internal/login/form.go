package login

import (
	"errors"
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

type DoneFn = func(token string)

type Form struct {
	*tview.Pages
	form *tview.Form
	app  *tview.Application
	done DoneFn
}

func NewForm(app *tview.Application, done DoneFn) *Form {
	self := &Form{
		Pages: tview.NewPages(),
		form:  tview.NewForm(),
		app:   app,
		done:  done,
	}

	self.
		form.
		AddInputField("Email", "", 0, nil, nil).
		AddPasswordField("Password", "", 0, 0, nil).
		AddPasswordField("Code (optional)", "", 0, 0, nil).
		AddButton("Login", self.login)
	self.AddAndSwitchToPage("form", self.form, true)
	return self
}

func (self *Form) login() {
	email := self.form.GetFormItem(0).(*tview.InputField).GetText()
	password := self.form.GetFormItem(1).(*tview.InputField).GetText()
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
		code := self.form.GetFormItem(2).(*tview.InputField).GetText()
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
		self.done(resp.Token)
	}
}

func (self *Form) onError(err error) {
	slog.Error("failed to login", "err", err)

	modal := tview.NewModal().
		SetText(err.Error()).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(_ int, _ string) {
			self.RemovePage("modal").SwitchToPage("form")
		})
	self.
		AddAndSwitchToPage("modal", centered(modal, 0, 0), true).
		ShowPage("form")
}

func centered(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}
