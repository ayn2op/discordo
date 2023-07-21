package ui

import (
	"log"

	"github.com/ayn2op/discordo/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

type LoginForm struct {
	*tview.Form
	Token chan string
}

func NewLoginForm() *LoginForm {
	lf := &LoginForm{
		Form:  tview.NewForm(),
		Token: make(chan string, 1),
	}

	lf.AddInputField("Email", "", 0, nil, nil)
	lf.AddPasswordField("Password", "", 0, 0, nil)
	lf.AddPasswordField("Code (optional)", "", 0, 0, nil)
	lf.AddCheckbox("Remember Me", true, nil)
	lf.AddButton("Login", lf.onLoginButtonSelected)

	lf.SetTitle("Login")
	lf.SetTitleColor(tcell.GetColor(config.Current.Theme.TitleColor))
	lf.SetTitleAlign(tview.AlignLeft)

	p := config.Current.Theme.BorderPadding
	lf.SetBorder(config.Current.Theme.Border)
	lf.SetBorderColor(tcell.GetColor(config.Current.Theme.BorderColor))
	lf.SetBorderPadding(p[0], p[1], p[2], p[3])

	return lf
}

func (lf *LoginForm) onLoginButtonSelected() {
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
		log.Fatal(err)
	}

	// If the account has MFA-enabled, attempt to log in using the provided code.
	if lr.MFA && lr.Token == "" {
		code := lf.GetFormItem(2).(*tview.InputField).GetText()
		if code == "" {
			return
		}

		lr, err = apiClient.TOTP(code, lr.Ticket)
		if err != nil {
			log.Fatal(err)
		}
	}

	if lr.Token == "" {
		log.Fatal("missing token")
	} else {
		rememberMe := lf.GetFormItem(3).(*tview.Checkbox).IsChecked()
		if rememberMe {
			go keyring.Set(config.Name, "token", lr.Token)
		}

		lf.Token <- lr.Token
	}
}
