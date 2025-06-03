package login

import (
	"errors"
	"log/slog"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
	"tui/internal/config"
	"tui/internal/consts"
	"tui/internal/ui"
)

type DoneFn = func(token string)

type Form struct {
	*tview.Pages
	cfg  *config.Config
	app  *tview.Application
	form *tview.Form
	done DoneFn
}

func NewForm(cfg *config.Config, app *tview.Application, done DoneFn) *Form {
	f := &Form{
		Pages: tview.NewPages(),
		cfg:   cfg,
		app:   app,
		form:  tview.NewForm(),
		done:  done,
	}

	f.form.
		AddInputField("Email", "", 0, nil, nil).
		AddPasswordField("Password", "", 0, 0, nil).
		AddPasswordField("Code (optional)", "", 0, 0, nil).
		AddButton("Login", f.login)
	f.AddAndSwitchToPage("form", f.form, true)
	return f
}

func (f *Form) login() {
	email := f.form.GetFormItem(0).(*tview.InputField).GetText()
	password := f.form.GetFormItem(1).(*tview.InputField).GetText()
	if email == "" || password == "" {
		return
	}

	// Create an API client without an authentication token.
	client := api.NewClient("")
	// Spoof the user agent of a web browser.
	client.UserAgent = f.cfg.Identify.UserAgent

	resp, err := client.Login(email, password)
	if err != nil {
		f.onError(err)
		return
	}

	if resp.Token == "" && resp.MFA {
		code := f.form.GetFormItem(2).(*tview.InputField).GetText()
		if code == "" {
			f.onError(errors.New("code required"))
			return
		}

		// Attempt to login using the code.
		resp, err = client.TOTP(code, resp.Ticket)
		if err != nil {
			f.onError(err)
			return
		}
	}

	if resp.Token == "" {
		f.onError(errors.New("missing token"))
		return
	}

	go keyring.Set(consts.Name, "token", resp.Token)

	if f.done != nil {
		f.done(resp.Token)
	}
}

func (f *Form) onError(err error) {
	slog.Error("failed to login", "err", err)

	modal := tview.NewModal().
		SetText(err.Error()).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(_ int, _ string) {
			f.RemovePage("modal").SwitchToPage("form")
		})
	f.
		AddAndSwitchToPage("modal", ui.Centered(modal, 0, 0), true).
		ShowPage("form")
}
