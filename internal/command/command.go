package command

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/sahilm/fuzzy"
)

type Item struct {
	Name         string
	Usage        string
	Description  string
	Source       string
	Builtin      bool
	Expand       func(args string) string
	RequiredPerm discord.Permissions
}

type itemList []Item

func (l itemList) String(i int) string {
	return l[i].Name
}

func (l itemList) Len() int {
	return len(l)
}

var Builtins = []Item{
	{
		Name:        "shrug",
		Usage:       "[message]",
		Description: "Appends ¯\\_(ツ)_/¯ to your message",
		Source:      "Built-in",
		Builtin:     true,
		Expand: func(args string) string {
			if args == "" {
				return `¯\_(ツ)_/¯`
			}
			return args + ` ¯\_(ツ)_/¯`
		},
	},
	{
		Name:        "tableflip",
		Usage:       "[message]",
		Description: "Appends (╯°□°)╯︵ ┻━┻ to your message",
		Source:      "Built-in",
		Builtin:     true,
		Expand: func(args string) string {
			if args == "" {
				return `(╯°□°)╯︵ ┻━┻`
			}
			return args + ` (╯°□°)╯︵ ┻━┻`
		},
	},
	{
		Name:        "unflip",
		Usage:       "[message]",
		Description: "Appends ┬─┬ノ( º _ ºノ) to your message",
		Source:      "Built-in",
		Builtin:     true,
		Expand: func(args string) string {
			if args == "" {
				return `┬─┬ノ( º _ ºノ)`
			}
			return args + ` ┬─┬ノ( º _ ºノ)`
		},
	},
	{
		Name:        "me",
		Usage:       "<message>",
		Description: "Displays text with emphasis",
		Source:      "Built-in",
		Builtin:     true,
		Expand: func(args string) string {
			if args == "" {
				return ""
			}
			return "*" + args + "*"
		},
	},
	{
		Name:        "spoiler",
		Usage:       "<message>",
		Description: "Marks your message as a spoiler",
		Source:      "Built-in",
		Builtin:     true,
		Expand: func(args string) string {
			if args == "" {
				return ""
			}
			return "||" + args + "||"
		},
	},
	{
		Name:         "nick",
		Usage:        "[new_nickname]",
		Description:  "Changes your nickname on the current server",
		Source:       "Built-in",
		Builtin:      true,
		RequiredPerm: discord.PermissionChangeNickname,
	},
}

// ExpandBuiltin expands recognized built-in slash commands if matched.
func ExpandBuiltin(text string) (string, bool) {
	if !strings.HasPrefix(text, "/") {
		return text, false
	}
	parts := strings.SplitN(text[1:], " ", 2)
	name := parts[0]
	var args string
	if len(parts) > 1 {
		args = parts[1]
	}

	for _, cmd := range Builtins {
		if strings.EqualFold(cmd.Name, name) && cmd.Expand != nil {
			return cmd.Expand(args), true
		}
	}
	return text, false
}

// Search filters built-in and bot application commands based on query and permissions.
func Search(query string, channelID discord.ChannelID, userPerms discord.Permissions, botCmds []Item, limit int) []Item {
	if limit <= 0 {
		limit = 20
	}

	var candidates []Item
	isAdmin := userPerms.Has(discord.PermissionAdministrator)

	for _, b := range Builtins {
		if b.RequiredPerm != 0 && !isAdmin && !userPerms.Has(b.RequiredPerm) {
			continue
		}
		candidates = append(candidates, b)
	}

	for _, b := range botCmds {
		if b.RequiredPerm != 0 && !isAdmin && !userPerms.Has(b.RequiredPerm) {
			continue
		}
		candidates = append(candidates, b)
	}

	if query == "" {
		if len(candidates) > limit {
			return candidates[:limit]
		}
		return candidates
	}

	lowerQuery := strings.ToLower(query)
	var results []Item
	for _, c := range candidates {
		if strings.HasPrefix(strings.ToLower(c.Name), lowerQuery) {
			results = append(results, c)
			if len(results) >= limit {
				return results
			}
		}
	}

	matches := fuzzy.FindFrom(query, itemList(candidates))
	for _, m := range matches {
		item := candidates[m.Index]
		if !slices.ContainsFunc(results, func(it Item) bool { return it.Name == item.Name && it.Source == item.Source }) {
			results = append(results, item)
			if len(results) >= limit {
				return results
			}
		}
	}

	return results
}

type searchResponse struct {
	Applications []struct {
		ID   discord.AppID `json:"id"`
		Name string        `json:"name"`
	} `json:"applications"`
	ApplicationCommands []discord.Command `json:"application_commands"`
}

// FetchBotCommands queries Discord's channel application commands endpoint for slash commands.
func FetchBotCommands(ctx context.Context, client *api.Client, channelID discord.ChannelID, query string) ([]Item, error) {
	if client == nil || !channelID.IsValid() {
		return nil, nil
	}

	endpoint := fmt.Sprintf("https://discord.com/api/v9/channels/%s/application-commands/search?query=%s&limit=25&include_applications=true",
		channelID.String(), url.QueryEscape(query))

	var resp searchResponse
	if err := client.WithContext(ctx).RequestJSONInto(&resp, "GET", endpoint); err != nil {
		return nil, err
	}

	appNames := make(map[discord.AppID]string, len(resp.Applications))
	for _, app := range resp.Applications {
		appNames[app.ID] = app.Name
	}

	items := make([]Item, 0, len(resp.ApplicationCommands))
	for _, cmd := range resp.ApplicationCommands {
		if cmd.Type != 0 && cmd.Type != discord.ChatInputCommand {
			continue
		}
		source := appNames[cmd.AppID]
		if source == "" {
			source = "Bot"
		}

		var reqPerm discord.Permissions
		if cmd.DefaultMemberPermissions != nil {
			reqPerm = discord.Permissions(*cmd.DefaultMemberPermissions)
		}

		var usageParts []string
		for _, opt := range cmd.Options {
			usageParts = append(usageParts, "["+opt.Name()+"]")
		}

		items = append(items, Item{
			Name:         cmd.Name,
			Usage:        strings.Join(usageParts, " "),
			Description:  cmd.Description,
			Source:       source,
			Builtin:      false,
			RequiredPerm: reqPerm,
		})
	}

	return items, nil
}
