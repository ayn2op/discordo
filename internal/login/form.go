package login

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ayn2op/discordo/internal/consts"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
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
	f := &Form{
		Pages: tview.NewPages(),
		form:  tview.NewForm(),
		app:   app,
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
	client.UserAgent = consts.UserAgent

	body := httputil.WithJSONBody(struct {
		Email    string `json:"login"`
		Password string `json:"password"`
	}{email, password})

	var (
		resp *api.LoginResponse
		err  error
	)
	err = client.RequestJSON(&resp, http.MethodPost, api.EndpointLogin, body)
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
		AddAndSwitchToPage("modal", centered(modal, 0, 0), true).
		ShowPage("form")
}

func centered(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}
