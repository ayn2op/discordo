package core

import (
	"context"

	"github.com/ayntgl/discordo/config"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
)

type Core struct {
	config  *config.Config
	session *session.Session
}

func New(token string, cfg *config.Config) *Core {
	s := session.NewWithIdentifier(gateway.NewIdentifier(gateway.IdentifyCommand{
		Token: token,
		Properties: gateway.IdentifyProperties{
			OS:      cfg.Identify.Os,
			Browser: cfg.Identify.Browser,
			Device:  "",
		},
		Presence: gateway.DefaultPresence,
	}))

	return &Core{
		config:  cfg,
		session: s,
	}
}

func (c *Core) Run() error {
	c.session.AddHandler(c.onSessionReady)
	err := c.session.Open(context.Background())
	if err != nil {
		return err
	}

	return err
}

func (c *Core) onSessionReady(r *gateway.ReadyEvent) {
}
