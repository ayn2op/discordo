package main

import (
	"encoding/json"
	"os"

	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

const service = "discordo"

func main() {
	app := ui.NewApp()
	app.EnableMouse(config.General.Mouse)

	token := os.Getenv("DISCORDO_TOKEN")
	if token == "" {
		token, _ = keyring.Get(service, "token")
	}

	if token != "" {
		err := app.Connect(token)
		if err != nil {
			panic(err)
		}

		ui.DrawMainFlex(app)
		app.
			SetRoot(app.MainFlex, true).
			SetFocus(app.GuildsList)
	} else {
		app.LoginForm = ui.NewLoginForm(func() {
			onLoginFormLoginButtonSelected(app)
		}, false)
		app.SetRoot(app.LoginForm, true)
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func onLoginFormLoginButtonSelected(app *ui.App) {
	email := app.LoginForm.GetFormItem(0).(*tview.InputField).GetText()
	password := app.LoginForm.GetFormItem(1).(*tview.InputField).GetText()
	if email == "" || password == "" {
		return
	}

	// Login using the email and password
	lr, err := login(app.Session, email, password)
	if err != nil {
		panic(err)
	}

	if lr.Token != "" && !lr.MFA {
		app.
			SetRoot(app.MainFlex, true).
			SetFocus(app.GuildsList)

		err = app.Connect(lr.Token)
		if err != nil {
			panic(err)
		}

		go keyring.Set(service, "token", lr.Token)
	} else if lr.MFA {
		// The account has MFA enabled, reattempt login with code and ticket.
		app.LoginForm = ui.NewLoginForm(func() {
			code := app.LoginForm.GetFormItem(0).(*tview.InputField).GetText()
			if code == "" {
				return
			}

			lr, err = totp(app.Session, code, lr.Ticket)
			if err != nil {
				panic(err)
			}

			app.
				SetRoot(app.MainFlex, true).
				SetFocus(app.GuildsList)

			err = app.Connect(lr.Token)
			if err != nil {
				panic(err)
			}

			go keyring.Set(service, "token", lr.Token)
		}, true)

		app.SetRoot(app.LoginForm, true)
	}
}

type loginResponse struct {
	MFA    bool   `json:"mfa"`
	SMS    bool   `json:"sms"`
	Ticket string `json:"ticket"`
	Token  string `json:"token"`
}

func login(s *discordgo.Session, email string, password string) (*loginResponse, error) {
	data := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{email, password}
	resp, err := s.RequestWithBucketID(
		"POST",
		discordgo.EndpointLogin,
		data,
		discordgo.EndpointLogin,
	)
	if err != nil {
		return nil, err
	}

	var lr loginResponse
	err = json.Unmarshal(resp, &lr)
	if err != nil {
		return nil, err
	}

	return &lr, nil
}

func totp(s *discordgo.Session, code string, ticket string) (*loginResponse, error) {
	data := struct {
		Code   string `json:"code"`
		Ticket string `json:"ticket"`
	}{code, ticket}
	e := discordgo.EndpointAuth + "mfa/totp"
	resp, err := s.RequestWithBucketID("POST", e, data, e)
	if err != nil {
		return nil, err
	}

	var lr loginResponse
	err = json.Unmarshal(resp, &lr)
	if err != nil {
		return nil, err
	}

	return &lr, nil
}
