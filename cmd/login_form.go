package cmd

import (
	"errors"

	"github.com/ayn2op/discordo/internal/constants"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

type DoneFn func(token string, err error)

type LoginForm struct {
	*tview.Form
	done DoneFn
}

func NewLoginForm(done DoneFn) *LoginForm {
	lf := &LoginForm{
		Form: tview.NewForm(),
		done: done,
	}

	lf.AddInputField("Email", "", 0, nil, nil)
	lf.AddPasswordField("Password", "", 0, 0, nil)
	lf.AddPasswordField("Code (optional)", "", 0, 0, nil)
	lf.AddCheckbox("Remember Me", true, nil)
	lf.AddButton("Login", lf.login)

	lf.SetTitle("Login")
	lf.SetTitleColor(tcell.GetColor(cfg.Theme.TitleColor))
	lf.SetTitleAlign(tview.AlignLeft)

	p := cfg.Theme.BorderPadding
	lf.SetBorder(cfg.Theme.Border)
	lf.SetBorderColor(tcell.GetColor(cfg.Theme.BorderColor))
	lf.SetBorderPadding(p[0], p[1], p[2], p[3])

	return lf
}

func (lf *LoginForm) login() {
	email := lf.GetFormItem(0).(*tview.InputField).GetText()
	password := lf.GetFormItem(1).(*tview.InputField).GetText()
	if email == "" || password == "" {
		return
	}

	// Create a new API client without an authentication token.
	apiClient := api.NewClient("")
	// Log in using the provided email and password.
	lr, err := apiClient.Login(email, password)
	if err != nil {
		if lf.done != nil {
			lf.done("", err)
		}

		return
	}

	// If the account has MFA-enabled, attempt to log in using the provided code.
	if lr.MFA && lr.Token == "" {
		code := lf.GetFormItem(2).(*tview.InputField).GetText()
		if code == "" {
			if lf.done != nil {
				lf.done("", errors.New("code required"))
			}

			return
		}

		lr, err = apiClient.TOTP(code, lr.Ticket)
		if err != nil {
			if lf.done != nil {
				lf.done("", err)
			}

			return
		}
	}

	if lr.Token == "" {
		if lf.done != nil {
			lf.done("", errors.New("missing token"))
		}

		return
	}

	rememberMe := lf.GetFormItem(3).(*tview.Checkbox).IsChecked()
	if rememberMe {
		go func() {
			if err := keyring.Set(constants.Name, "token", lr.Token); err != nil {
				if lf.done != nil {
					lf.done("", err)
				}
			}
		}()
	}

	if lf.done != nil {
		lf.done(lr.Token, nil)
	}
}
