package http

import (
	stdHttp "net/http"
)

func Headers() stdHttp.Header {
	headers := make(stdHttp.Header)
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers
	headers.Set("Accept", "*/*")
	headers.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	headers.Set("Accept-Language", "en-US,en;q=0.9")
	headers.Set("Origin", "https://discord.com")
	headers.Set("Priority", "u=0, i")
	headers.Set("Referer", "https://discord.com/channels/@me")
	headers.Set("Sec-Fetch-Dest", "empty")
	headers.Set("Sec-Fetch-Mode", "cors")
	headers.Set("Sec-Fetch-Site", "same-origin")

	headers.Set("X-Debug-Options", "bugReporterEnabled")
	headers.Set("X-Discord-Locale", string(Locale))

	if superProps, err := superProps(); err == nil {
		headers.Set("X-Super-Properties", superProps)
	}

	return headers
}
