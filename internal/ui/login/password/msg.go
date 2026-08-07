package password

import (
	"errors"

	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/tview"
)

type (
	TokenMsg string
	ErrMsg   error
)

func loginCmd(login, password string) tview.Cmd {
	return func() tview.Msg {
		headers := http.Headers()
		headers.Set("Referer", "https://discord.com/login")

		client := http.NewClient("")
		client.OnRequest = append(client.OnRequest, httputil.WithHeaders(headers))

		loginResp, err := client.Login(login, password)
		if err != nil {
			return ErrMsg(err)
		}
		if loginResp.Token == "" {
			if loginResp.MFA {
				return ErrMsg(errors.New("multi-factor authentication is required; use token or QR login"))
			}
			return ErrMsg(errors.New("login response did not include a token"))
		}
		return TokenMsg(loginResp.Token)
	}
}
