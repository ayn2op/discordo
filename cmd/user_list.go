package cmd

import (
	"log"
	"fmt"

	"github.com/ayn2op/discordo/internal/constants"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type UserList struct {
	*tview.TreeView

	root           *tview.TreeNode
	selectedUserID discord.UserID
}

func newUserList() *UserList {
	ul := &UserList{
		TreeView: tview.NewTreeView(),
		root: tview.NewTreeNode(""),
	}

	ul.SetTopLevel(1)
	ul.SetRoot(ul.root)
	ul.SetGraphics(true)
	ul.SetBackgroundColor(tcell.GetColor(cfg.Theme.BackgroundColor))
	ul.SetSelectedFunc(ul.onSelected)

	ul.SetTitle("Users")
	ul.SetTitleColor(tcell.GetColor(cfg.Theme.TitleColor))
	ul.SetTitleAlign(tview.AlignLeft)

	p := cfg.Theme.BorderPadding
	ul.SetBorder(cfg.Theme.Border)
	ul.SetBorderColor(tcell.GetColor(cfg.Theme.BorderColor))
	ul.SetBorderPadding(p[0], p[1], p[2], p[3])

	return ul
}

func (ul *UserList) createUserNode(user discord.User) {
	var username string
	if user.DisplayName != "" {
		username = user.DisplayName
	} else {
		username = user.Username
	}

	userNode := tview.NewTreeNode(username)
	userNode.SetReference(user.ID)
	ul.root.AddChild(userNode)
	// TODO: Add dropdown commands
}

func (ul *UserList) updateUsersFromGuildID(g discord.GuildID) {
	ul.root.ClearChildren()
	users, err := discordState.Members(g)
	if err != nil {
		log.Printf("Unable to find Guild with ID %s\n", g)
		return
	}
	for _, user := range users {
		ul.createUserNode(user.User)
	}
	ul.SetCurrentNode(ul.root)
}

func (ul *UserList) onSelected(n *tview.TreeNode) {
	switch ref := n.GetReference().(type) {
	case discord.UserID:
		n.ClearChildren()
		n.AddChild(tview.NewTreeNode(constants.UserListCmdMention))
		n.SetExpanded(!n.IsExpanded())
		return
	case nil: // Dropdown options/commands
		switch n.GetText() {
		case constants.UserListCmdMention:
			mainFlex.messageInput.SetText(fmt.Sprintf("User ID: %d", ref), true)
		}

		// TODO: run user command based on node name
		return
	}
}
