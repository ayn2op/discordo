package gateway

import (
	"testing"

	discordgateway "github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/utils/ws"
)

func TestNew(t *testing.T) {
	t.Run("authentication failed", func(t *testing.T) {
		opts := New(discordgateway.DefaultIdentifier("token")).Opts()
		if !opts.ErrorIsFatalClose(&ws.CloseEvent{Code: 4004}) {
			t.Error("gateway reconnects after the token is rejected")
		}
	})
}
