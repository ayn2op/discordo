package login

import (
	"errors"
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"golang.design/x/clipboard"
)

const (
	formPageName  = "form"
	errorPageName = "error"
	qrPageName    = "qr"
)

type DoneFn = func(token string)

type Form struct {
	*tview.Pages
	app  *tview.Application
	cfg  *config.Config
	form *tview.Form
	done DoneFn
}

func NewForm(app *tview.Application, cfg *config.Config, done DoneFn) *Form {
	f := &Form{
		Pages: tview.NewPages(),
		app:   app,
		cfg:   cfg,
		form:  tview.NewForm(),
		done:  done,
	}

	f.form.
		AddInputField("Email", "", 0, nil).
		AddPasswordField("Password", "", 0, 0, nil).
		AddPasswordField("Code (optional)", "", 0, 0, nil).
		AddButton("Login", f.login).
		AddButton("Login with QR", f.loginWithQR)
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
	props := http.IdentifyProperties()
	if browserUserAgent, ok := props["browser_user_agent"]; ok {
		if val, ok := browserUserAgent.(string); ok {
			api.UserAgent = val
		}
	}

	resp, err := client.Login(email, password)
	if err != nil {
		f.onError(err)
		return
	}

	if resp.MFA {
		switch {
		case resp.TOTP:
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
		default:
			f.onError(errors.New("unsupported mfa type"))
			return
		}
	}

	if resp.Token == "" {
		f.onError(errors.New("missing token"))
		return
	}

	go keyring.SetToken(resp.Token)

	if f.done != nil {
		f.done(resp.Token)
	}
}

func (f *Form) onError(err error) {
	slog.Error("failed to login", "err", err)

	message := err.Error()
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Copy", "Close"}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			if buttonIndex == 0 {
				go clipboard.Write(clipboard.FmtText, []byte(message))
			} else {
				f.
					RemovePage(errorPageName).
					SwitchToPage(formPageName)
			}
		})
	f.
		AddAndSwitchToPage(errorPageName, ui.Centered(modal, 0, 0), true).
		ShowPage(formPageName)
}

func (f *Form) loginWithQR() {
	qr := newQRLogin(f.app, f.cfg, func(token string, err error) {
		if err != nil {
			f.onError(err)
			return
		}

		if token == "" {
			f.RemovePage(qrPageName).SwitchToPage(formPageName)
			return
		}

		go keyring.SetToken(token)

		f.RemovePage(qrPageName)
		if f.done != nil {
			f.done(token)
		}
	})

	f.AddAndSwitchToPage(qrPageName, qr, true)
	qr.start()
}
