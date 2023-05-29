package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type GuildsTree struct {
	*tview.TreeView

	root              *tview.TreeNode
	selectedChannelID discord.ChannelID
}

func newGuildsTree() *GuildsTree {
	gt := &GuildsTree{
		TreeView: tview.NewTreeView(),

		root: tview.NewTreeNode(""),
	}

	gt.SetTopLevel(1)
	gt.SetRoot(gt.root)
	gt.SetGraphics(config.Current.Theme.GuildsTree.Graphics)
	gt.SetBackgroundColor(tcell.GetColor(config.Current.Theme.BackgroundColor))
	gt.SetSelectedFunc(gt.onSelected)

	gt.SetTitle("Guilds")
	gt.SetTitleColor(tcell.GetColor(config.Current.Theme.TitleColor))
	gt.SetTitleAlign(tview.AlignLeft)

	p := config.Current.Theme.BorderPadding
	gt.SetBorder(config.Current.Theme.Border)
	gt.SetBorderColor(tcell.GetColor(config.Current.Theme.BorderColor))
	gt.SetBorderPadding(p[0], p[1], p[2], p[3])

	return gt
}

func (gt *GuildsTree) createGuildFolderNode(parent *tview.TreeNode, gf gateway.GuildFolder) {
	var name string
	if gf.Name != "" {
		name = fmt.Sprintf("[%s]%s[-]", gf.Color.String(), gf.Name)
	} else {
		name = "Folder"
	}

	n := tview.NewTreeNode(name)
	parent.AddChild(n)

	for _, gid := range gf.GuildIDs {
		g, err := discordState.Cabinet.Guild(gid)
		if err != nil {
			log.Println(err)
			continue
		}

		gt.createGuildNode(n, *g)
	}
}

func (gt *GuildsTree) createGuildNode(n *tview.TreeNode, g discord.Guild) {
	readyExtras := discordState.Ready().ReadyEventExtras
	tag := "[::]"
	
	// Set the tag for the unread/muted/pinged indicators
	//
	// If the guild is muted, then we can safely just move to the 
	// node creation and call it a day
	//
	// You'll also notice that we go to node creation if an error
	// occurs. This is because this only relates to unread and read, so
	// if an error occurs, the user should still be able to view the channel safely
	{
		var guildSettings *gateway.UserGuildSetting
		unreadCount := 0
		mentionCount := 0
		for ig := range readyExtras.UserGuildSettings {
			if readyExtras.UserGuildSettings[ig].GuildID == g.ID {
				guildSettings = &readyExtras.UserGuildSettings[ig]
				if guildSettings.Muted {
					tag = fmt.Sprintf("[%s]", config.Current.Theme.GuildsTree.MutedIndicator)
					goto create_guild_node
				}
			}
		}
		
		// Turns out that ReadyEventExtras is not required
		// to have all guilds, so we're going to have to 
		// handle that seperately
		if guildSettings != nil {
			for ic := range guildSettings.ChannelOverrides {
				channelSettings := guildSettings.ChannelOverrides[ic]
				channel, err := discordState.Channel(channelSettings.ChannelID)
				if err != nil {
					log.Println(err)
					continue
				} else if channelSettings.Muted {
					continue
				}
				for rs := range readyExtras.ReadStates {
					readState := readyExtras.ReadStates[rs]
					if readState.ChannelID == channel.ID {
						if readState.LastMessageID != channel.LastMessageID {
							unreadCount += 1
						}
						if readState.MentionCount > 0 {
							mentionCount += readState.MentionCount
						}
						break
					}
				}
			} 
		}

		// Get all non-overriden channels, cause 
		// Discord sucks at their API
		channels, err := discordState.Channels(g.ID)
		if err != nil {
			log.Println(err)
			goto create_guild_node
		}
		
		// "Wait, didn't we already do this?"
		//
		// Well not quite. ChannelOverrides only 
		// contains overriden channels, so we have to
		// handle the rest of them
		//
		// Yay...
		for ic := range channels {
			for rs := range readyExtras.ReadStates {
				readState := readyExtras.ReadStates[rs]
				if readState.ChannelID == channels[ic].ID {
					if readState.LastMessageID != channels[ic].LastMessageID {
						unreadCount += 1
					}
					if readState.MentionCount > 0 {
						mentionCount += readState.MentionCount
					}
					break
				}
			}
		}

		if unreadCount > 0 { 
			tag = fmt.Sprintf("[%s]", config.Current.Theme.GuildsTree.UnreadIndicator)
		} 
		if mentionCount > 0 {
			tag = fmt.Sprintf("[%s]", 
				fmt.Sprintf("%s%s", 
					config.Current.Theme.GuildsTree.MentionColor, 
					config.Current.Theme.GuildsTree.UnreadIndicator)) + fmt.Sprint(mentionCount) + " "
		}
	}
	
	create_guild_node:
	gn := tview.NewTreeNode(tag + g.Name + "[::-]")
	gn.SetReference(g.ID)
	n.AddChild(gn)
}

func (gt *GuildsTree) channelToString(c discord.Channel) string {
	var s string
	tag := "[::]"
	readyExtras := discordState.Ready().ReadyEventExtras
	var channelSettings gateway.UserChannelOverride
	var guildSettings gateway.UserGuildSetting
	
	// Get the guild and channel settings so we can respect
	// whether the user wants unread indicators or not
	//
	// We use seperate for loops to avoid heavy nesting, and gotos
	// to avoid doing extra loops if unnecesary
	{
		// Get guild settings
		for ig := range readyExtras.UserGuildSettings {
			if readyExtras.UserGuildSettings[ig].GuildID == c.GuildID {
				guildSettings = readyExtras.UserGuildSettings[ig]
				break	
			}
		}
		// Get channel settings
		for ico := range guildSettings.ChannelOverrides {
			if guildSettings.ChannelOverrides[ico].ChannelID == c.ID {
				channelSettings = guildSettings.ChannelOverrides[ico]
				if channelSettings.Muted {
					tag = fmt.Sprintf("[%s]", config.Current.Theme.GuildsTree.MutedIndicator)
					break
				}
				break
			}
		}
	}

	// Go over each of the read states, then if:
	// - The channel IDs match
	// - The last message IDs don't match
	// then set the tag to bold and italics
	//
	// Pings are exempt from channels being muted, since
	// pings are important and as such should be seen regardless
	for ic := range readyExtras.ReadStates {
		if readyExtras.ReadStates[ic].ChannelID == c.ID && readyExtras.ReadStates[ic].LastMessageID != c.LastMessageID {
			mentionCount := readyExtras.ReadStates[ic].MentionCount
			if mentionCount > 0 {
				tag = fmt.Sprintf("[%s]", 
					fmt.Sprintf("%s%s", 
						config.Current.Theme.GuildsTree.MentionColor, 
						config.Current.Theme.GuildsTree.UnreadIndicator)) + fmt.Sprint(mentionCount)
			} else if mentionCount == 0 && !channelSettings.Muted {
				tag = fmt.Sprintf("[%s]", config.Current.Theme.GuildsTree.UnreadIndicator)
			}
			break
		}
	}
	
	switch c.Type {
	case discord.GuildText:
		s = "#" + c.Name
	case discord.DirectMessage:
		r := c.DMRecipients[0]
		s = r.Tag()
	case discord.GuildVoice:
		s = "v-" + c.Name
	case discord.GroupDM:
		s = c.Name
		// If the name of the channel is empty, use the recipients' tags
		if s == "" {
			rs := make([]string, len(c.DMRecipients))
			for _, r := range c.DMRecipients {
				rs = append(rs, r.Tag())
			}

			s = strings.Join(rs, ", ")
		}
	case discord.GuildAnnouncement:
		s = "a-" + c.Name
	case discord.GuildStore:
		s = "s-" + c.Name
	case discord.GuildForum:
		s = "f-" + c.Name
	default:
		s = c.Name
	}

	return tag + s + "[::-]"
}

func (gt *GuildsTree) createChannelNode(n *tview.TreeNode, c discord.Channel) {
	if c.Type != discord.DirectMessage && c.Type != discord.GroupDM {
		ps, err := discordState.Permissions(c.ID, discordState.Ready().User.ID)
		if err != nil {
			log.Println(err)
			return
		}

		if !ps.Has(discord.PermissionViewChannel) {
			return
		}
	}

	cn := tview.NewTreeNode(gt.channelToString(c))
	cn.SetReference(c.ID)
	n.AddChild(cn)
}

func (gt *GuildsTree) createChannelNodes(n *tview.TreeNode, cs []discord.Channel) {
	for _, c := range cs {
		if c.Type != discord.GuildCategory && !c.ParentID.IsValid() {
			gt.createChannelNode(n, c)
		}
	}

PARENT_CHANNELS:
	for _, c := range cs {
		if c.Type == discord.GuildCategory {
			for _, nested := range cs {
				if nested.ParentID == c.ID {
					gt.createChannelNode(n, c)
					continue PARENT_CHANNELS
				}
			}
		}
	}

	for _, c := range cs {
		if c.ParentID.IsValid() {
			var parent *tview.TreeNode
			n.Walk(func(node, _ *tview.TreeNode) bool {
				if node.GetReference() == c.ParentID {
					parent = node
					return false
				}

				return true
			})

			if parent != nil {
				gt.createChannelNode(parent, c)
			}
		}
	}
}

func (gt *GuildsTree) onSelected(n *tview.TreeNode) {
	gt.selectedChannelID = 0

	mainFlex.messagesText.reset()
	mainFlex.messageInput.reset()

	if len(n.GetChildren()) != 0 {
		n.SetExpanded(!n.IsExpanded())
		return
	}

	switch ref := n.GetReference().(type) {
	case discord.GuildID:
		cs, err := discordState.Cabinet.Channels(ref)
		if err != nil {
			log.Println(err)
			return
		}

		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		gt.createChannelNodes(n, cs)
	case discord.ChannelID:
		ms, err := discordState.Messages(ref, config.Current.MessagesLimit)
		if err != nil {
			log.Println(err)
			return
		}

		for i := len(ms) - 1; i >= 0; i-- {
			mainFlex.messagesText.createMessage(ms[i])
		}

		c, err := discordState.Cabinet.Channel(ref)
		if err != nil {
			log.Println(err)
			return
		}

		mainFlex.messagesText.SetTitle(gt.channelToString(*c))

		gt.selectedChannelID = ref
		app.SetFocus(mainFlex.messageInput)
	case nil: // Direct messages
		cs, err := discordState.Cabinet.PrivateChannels()
		if err != nil {
			log.Println(err)
			return
		}

		sort.Slice(cs, func(i, j int) bool {
			return cs[i].LastMessageID > cs[j].LastMessageID
		})

		for _, c := range cs {
			gt.createChannelNode(n, c)
		}
	}
}
