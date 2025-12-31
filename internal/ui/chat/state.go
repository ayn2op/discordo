package chat

import (
	"context"
	"log/slog"

	"github.com/ayn2op/discordo/internal/http"
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

func (cv *ChatView) OpenState(token string) error {
	identifyProps := http.IdentifyProperties()
	gateway.DefaultIdentity = identifyProps
	gateway.DefaultPresence = &gateway.UpdatePresenceCommand{
		Status: cv.cfg.Status,
	}

	id := gateway.DefaultIdentifier(token)
	id.Compress = false

	session := session.NewCustom(id, http.NewClient(token), handler.New())
	state := state.NewFromSession(session, defaultstore.New())
	cv.state = ningen.FromState(state)

	// Handlers
	cv.state.AddHandler(cv.onRaw)
	cv.state.AddHandler(cv.onReady)
	cv.state.AddHandler(cv.onMessageCreate)
	cv.state.AddHandler(cv.onMessageUpdate)
	cv.state.AddHandler(cv.onMessageDelete)
	cv.state.AddHandler(cv.onReadUpdate)
	cv.state.AddHandler(cv.onGuildMembersChunk)
	cv.state.AddHandler(cv.onGuildMemberRemove)

	cv.state.StateLog = func(err error) {
		slog.Error("state log", "err", err)
	}

	cv.state.OnRequest = append(cv.state.OnRequest, httputil.WithHeaders(http.Headers()), cv.onRequest)
	return cv.state.Open(context.TODO())
}

func (cv *ChatView) CloseState() error {
	if cv.state == nil {
		return nil
	}
	return cv.state.Close()
}

func (cv *ChatView) onRequest(r httpdriver.Request) error {
	if req, ok := r.(*httpdriver.DefaultRequest); ok {
		slog.Debug("new HTTP request", "method", req.Method, "url", req.URL)
	}

	return nil
}

func (cv *ChatView) onRaw(event *ws.RawEvent) {
	slog.Debug(
		"new raw event",
		"code", event.OriginalCode,
		"type", event.OriginalType,
		// "data", event.Raw,
	)
}
