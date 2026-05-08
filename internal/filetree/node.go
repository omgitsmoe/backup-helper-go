package filetree

import (
	"slices"
	"strings"
)

type Node struct {
	Name string
	parent *Node
	IsDir bool
	firstChild *Node
	nextSibling *Node
}

func NewNode(name string, isdir bool, parent *Node) *Node {
	return &Node{
		Name: name,
		parent: parent,
		IsDir: isdir,
	}
}

func (n *Node) AddChild(child *Node) {
	child.parent = n

	if n.firstChild != nil {
		n.firstChild.AddSibling(child)
	} else {
		n.firstChild = child
	}
}

func (n *Node) AddSibling(sibling *Node) {
	c := n
	for {
		if c.nextSibling != nil {
			c = c.nextSibling
		} else {
			c.nextSibling = sibling
			break
		}
	}
}

func (n *Node) Path() string {
	curr := n
	components := []string{}
	for curr != nil {
		components = append(components, curr.Name)
		curr = curr.parent
	}

	slices.Reverse(components)

	return strings.Join(components, "/")
}
