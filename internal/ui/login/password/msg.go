package password

import (
	"errors"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/tview"
)

type TokenMsg string

// inputMsg is the text and cursor that typing in an input left.
type inputMsg struct {
	text   string
	cursor int
}

type (
	loginMsg    inputMsg
	passwordMsg inputMsg
	codeMsg     inputMsg
	focusMsg    focus
	hoverMsg    bool
	submitMsg   struct{}
)

// mfaMsg carries the ticket Discord returned so the code entered next can complete the login.
type mfaMsg struct {
	method          mfaMethod
	ticket          string
	loginInstanceID string
}

// loginClient builds the REST client used for the unauthenticated login requests.
func loginClient() *api.Client {
	headers := http.Headers()
	headers.Set("Referer", "https://discord.com/login")

	client := http.NewClient("")
	client.OnRequest = append(client.OnRequest, httputil.WithHeaders(headers))
	return client
}

func submitLogin(login, password string) tview.Cmd {
	return func() tview.Msg {
		client := loginClient()

		loginResp, err := client.Login(login, password)
		if err != nil {
			return err
		}
		if loginResp.Token != "" {
			return TokenMsg(loginResp.Token)
		}
		if !loginResp.MFA {
			return errors.New("login response did not include a token")
		}

		if loginResp.SMS && !loginResp.TOTP {
			if _, err := client.SendMFASMS(loginResp.Ticket); err != nil {
				return err
			}
			return mfaMsg{mfaMethodSMS, loginResp.Ticket, loginResp.LoginInstanceID}
		}
		// Either an authenticator is set up, or only a backup code is available; both go through the TOTP endpoint.
		return mfaMsg{mfaMethodTOTP, loginResp.Ticket, loginResp.LoginInstanceID}
	}
}

func verifyMFACode(method mfaMethod, code, ticket, loginInstanceID string) tview.Cmd {
	return func() tview.Msg {
		client := loginClient()

		verify := client.TOTP
		if method == mfaMethodSMS {
			verify = client.SMS
		}

		loginResp, err := verify(code, ticket, loginInstanceID)
		if err != nil {
			return err
		}
		if loginResp.Token == "" {
			return errors.New("login response did not include a token")
		}
		return TokenMsg(loginResp.Token)
	}
}
