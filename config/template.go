package config

const defaultTemplate = "{{/* Learn more about templates: https://pkg.go.dev/text/template */}}" +
	// Define a new region and assign message ID as the region ID.
	// Learn more: https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
	"[\"{{.Message.ID}}\"]" +
	// Build the message associated with crosspost, channel follow add, pin, or a reply.
	"{{if (.Message.ReferencedMessage)}}" +
	" ╭ " +
	// Build the author of this message.
	"{{if (eq .Message.Author.ID .UserID)}}" +
	"[#57F287]{{.Message.Author.Username}}[-]" +
	"{{else}}" +
	"[#ED4245]{{.Message.Author.Username}}[-]" +
	"{{end}}" +
	// If the message author is a bot account, render the message with "BOT" label for distinction.
	"{{if .Message.Author.Bot}}" +
	" [#EB459E]BOT[-]" +
	"{{end}} " +

	"[::d]{{.Message.ReferencedMessage.Content}}[::-]\n" +
	"{{end}}" +

	// Build the author of this message.
	"{{if (eq .Message.Author.ID .UserID)}}" +
	"[#57F287]{{.Message.Author.Username}}[-]" +
	"{{ else }}" +
	"[#ED4245]{{.Message.Author.Username}}[-]" +
	"{{end}}" +
	// If the message author is a bot account, render the message with "BOT" label for distinction.
	"{{if .Message.Author.Bot}}" +
	" [#EB459E]BOT[-]" +
	"{{end}} " +

	"{{.Timestamp}}\n\t" +
	"{{.Content}}" +

	"{{if (.Message.EditedTimestamp.IsValid)}}" +
	" [::d](edited)[::-]" +
	"{{end}}" +

	// Tags with no region ID ([""]) do not start new regions. They can
	// therefore be used to mark the end of a region.
	"[\"\"]\n\n"
