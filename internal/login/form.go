package login

import (
	"errors"
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/zalando/go-keyring"
)

const (
	formPageName  = "form"
	errorPageName = "error"
)

type DoneFn = func(token string)

type Form struct {
	*tview.Pages
	cfg  *config.Config
	form *tview.Form
	done DoneFn
}

func NewForm(cfg *config.Config, done DoneFn) *Form {
	f := &Form{
		Pages: tview.NewPages(),
		cfg:   cfg,
		form:  tview.NewForm(),
		done:  done,
	}

	f.form.
		AddInputField("Email", "", 0, nil, nil).
		AddPasswordField("Password", "", 0, 0, nil).
		AddPasswordField("Code (optional)", "", 0, 0, nil).
		AddButton("Login", f.login)
	f.AddAndSwitchToPage(formPageName, f.form, true)
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
	props, err := consts.GetIdentifyProperties()
	if err != nil {
		f.onError(err)
		return
	}

	client.UserAgent = props.BrowserUserAgent

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
			f.
				RemovePage(errorPageName).
				SwitchToPage(formPageName)
		})
	f.
		AddAndSwitchToPage(errorPageName, ui.Centered(modal, 0, 0), true).
		ShowPage(formPageName)
}
