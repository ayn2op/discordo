package chat

import (
	"context"
	"log/slog"
	stdhttp "net/http"

	"github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/discordo/internal/profile"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/state/store/defaultstore"
	"github.com/diamondburned/arikawa/v3/utils/handler"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/ningen/v3"
)

func (v *View) OpenState(token string) error {
	identifyProps := http.IdentifyProperties()
	gateway.DefaultIdentity = identifyProps
	gateway.DefaultPresence = &gateway.UpdatePresenceCommand{
		Status: v.cfg.Status,
	}

	id := gateway.DefaultIdentifier(token)
	id.Compress = false

	session := session.NewCustom(id, http.NewClient(token), handler.New())
	state := state.NewFromSession(session, defaultstore.New())
	v.state = ningen.FromState(state)

	httpClient := &stdhttp.Client{Transport: http.NewTransport()}
	v.profileCache = profile.NewCache(httpClient, token)

	// Handlers
	v.state.AddHandler(v.onRaw)
	v.state.AddHandler(v.onReady)
	v.state.AddHandler(v.onMessageCreate)
	v.state.AddHandler(v.onMessageUpdate)
	v.state.AddHandler(v.onMessageDelete)
	v.state.AddHandler(v.onReadUpdate)
	v.state.AddHandler(v.onGuildMembersChunk)
	v.state.AddHandler(v.onGuildMemberRemove)

	v.state.StateLog = func(err error) {
		slog.Error("state log", "err", err)
	}

	v.state.OnRequest = append(v.state.OnRequest, httputil.WithHeaders(http.Headers()), v.onRequest)
	return v.state.Open(context.TODO())
}

func (v *View) CloseState() error {
	if v.state == nil {
		return nil
	}
	return v.state.Close()
}

func (v *View) onRequest(r httpdriver.Request) error {
	if req, ok := r.(*httpdriver.DefaultRequest); ok {
		slog.Debug("new HTTP request", "method", req.Method, "url", req.URL)
	}

	return nil
}

func (v *View) onRaw(event *ws.RawEvent) {
	slog.Debug(
		"new raw event",
		"code", event.OriginalCode,
		"type", event.OriginalType,
		// "data", event.Raw,
	)
}
