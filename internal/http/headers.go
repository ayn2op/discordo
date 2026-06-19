package http

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	stdHttp "net/http"

	"github.com/ayn2op/arikawa/v3/api"
)

func Headers() stdHttp.Header {
	headers := make(stdHttp.Header)
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers
	headers.Set("Accept", "*/*")
	headers.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	headers.Set("Accept-Language", "en-US,en;q=0.9")
	headers.Set("Origin", api.BaseEndpoint)
	headers.Set("Priority", "u=1, i")
	headers.Set("Referer", "https://discord.com/channels/@me")

	headers.Set("Sec-Fetch-Dest", "empty")
	headers.Set("Sec-Fetch-Mode", "cors")
	headers.Set("Sec-Fetch-Site", "same-origin")

	headers.Set("X-Debug-Options", "bugReporterEnabled")
	headers.Set("X-Discord-Locale", string(Locale))

	superProps, err := json.Marshal(XSuperProperties())
	if err != nil {
		slog.Error("failed to marshal super props", "err", err)
	} else {
		headers.Set("X-Super-Properties", base64.StdEncoding.EncodeToString(superProps))
	}

	return headers
}
