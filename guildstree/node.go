package guildstree

import (
	"github.com/ayn2op/discordo/tree"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
)

type baseNode struct {
	children []tree.Node
	expanded bool
}

func (bn baseNode) Children() []tree.Node {
	return bn.children
}

func (bn baseNode) Expanded() bool {
	return bn.expanded
}

func (bn *baseNode) SetExpanded(value bool) {
	bn.expanded = value
}

func (bn *baseNode) AppendChild(child tree.Node) {
	bn.children = append(bn.children, child)
}

type folderNode struct {
	baseNode
	gateway.GuildFolder
}

func (fn folderNode) Name() string {
	name := fn.GuildFolder.Name
	if name == "" {
		name = "Folder"
	}

	return name
}

type guildNode struct {
	baseNode
	discord.Guild
}

func (gn guildNode) Name() string {
	return gn.Guild.Name
}
