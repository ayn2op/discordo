package login

import (
	"errors"
	"log/slog"

	"github.com/ayn2op/tview/layers"
	"github.com/gdamore/tcell/v3"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"golang.design/x/clipboard"
)

const (
	formLayerName  = "form"
	errorLayerName = "error"
	qrLayerName    = "qr"
)

type DoneFn = func(token string)

type Form struct {
	*layers.Layers
	app  *tview.Application
	cfg  *config.Config
	form *tview.Form
	done DoneFn
}

func NewForm(app *tview.Application, cfg *config.Config, done DoneFn) *Form {
	f := &Form{
		Layers: layers.New(),
		app:    app,
		cfg:    cfg,
		form:   tview.NewForm(),
		done:   done,
	}

	f.form.
		AddPasswordField("Token", "", 0, 0, nil).
		AddButton("Login", f.login).
		AddButton("Login with QR", f.loginWithQR)
	f.SetBackgroundLayerStyle(f.cfg.Theme.Dialog.BackgroundStyle.Style)
	f.AddLayer(f.form, layers.WithName(formLayerName), layers.WithResize(true), layers.WithVisible(true))
	return f
}

func (f *Form) login() {
	token := f.form.GetFormItem(0).(*tview.InputField).GetText()
	if token == "" {
		f.onError(errors.New("token required"))
		return
	}

	go keyring.SetToken(token)

	if f.done != nil {
		f.done(token)
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
				f.RemoveLayer(errorLayerName)
			}
		})
	{
		bg := f.cfg.Theme.Dialog.Style.GetBackground()
		buttonStyle := f.cfg.Theme.Dialog.Style.Style
		if bg != tcell.ColorDefault {
			modal.SetBackgroundColor(bg)
			buttonStyle = buttonStyle.Background(bg)
		}
		fg := f.cfg.Theme.Dialog.Style.GetForeground()
		if fg != tcell.ColorDefault {
			modal.SetTextColor(fg)
			buttonStyle = buttonStyle.Foreground(fg)
		}
		// Keep button styles aligned with dialog content without hiding text.
		modal.SetButtonStyle(buttonStyle)
		modal.SetButtonActivatedStyle(buttonStyle)
	}
	f.
		AddLayer(
			ui.Centered(modal, 0, 0),
			layers.WithName(errorLayerName),
			layers.WithResize(true),
			layers.WithVisible(true),
			layers.WithOverlay(),
		).
		SendToFront(errorLayerName)
}

func (f *Form) loginWithQR() {
	qr := newQRLogin(f.app, f.cfg, func(token string, err error) {
		if err != nil {
			f.onError(err)
			return
		}

		if token == "" {
			f.RemoveLayer(qrLayerName)
			return
		}

		go keyring.SetToken(token)

		f.RemoveLayer(qrLayerName)
		if f.done != nil {
			f.done(token)
		}
	})

	f.AddLayer(qr, layers.WithName(qrLayerName), layers.WithResize(true), layers.WithVisible(true))
	qr.start()
}
