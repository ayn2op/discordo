package login

import (
	"errors"
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"golang.design/x/clipboard"
)

const (
	formPageName  = "form"
	errorPageName = "error"
	qrPageName    = "qr"
)

type DoneFn = func(token string) error

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
		AddPasswordField("Token", "", 0, 0, nil).
		AddButton("Login", f.login).
		AddButton("Login with QR", f.loginWithQR)
	f.AddAndSwitchToPage(formPageName, f.form, true)
	return f
}

func (f *Form) login() {
	token := f.form.GetFormItem(0).(*tview.InputField).GetText()
	if token == "" {
		f.ShowError(errors.New("token required"))
		return
	}

	if f.done != nil {
		if err := f.done(token); err != nil {
			f.ShowError(err)
		} else {
			go keyring.SetToken(token)
		}
	}
}

func (f *Form) ShowError(err error) {
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
			f.ShowError(err)
			return
		}

		if token == "" {
			f.RemovePage(qrPageName).SwitchToPage(formPageName)
			return
		}

		f.RemovePage(qrPageName)
		if f.done != nil {
			if err := f.done(token); err != nil {
				f.ShowError(err)
			} else {
				go keyring.SetToken(token)
			}
		}
	})

	f.AddAndSwitchToPage(qrPageName, qr, true)
	qr.start()
}
