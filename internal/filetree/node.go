package filetree

import (
	"strings"
)

func NewNode(name string, isdir bool, parent *Node) *Node {
	return &Node{
		Name: name,
		parent: parent,
		IsDir: isdir,
	}
}

type Node struct {
	Name  string
	parent *Node
	IsDir bool

	firstChild  *Node
	lastChild   *Node
	nextSibling *Node

	Index int // directory index, or -1 for files/root if you want
}

func (n *Node) AddChild(child *Node) {
	child.parent = n

	if n.firstChild == nil {
		n.firstChild = child
		n.lastChild = child
		return
	}

	n.lastChild.nextSibling = child
	n.lastChild = child
}

func (n *Node) Path() string {
	curr := n
	components := make([]string, 0, 8)

	for curr != nil {
		if curr.Name != "" {
			components = append(components, curr.Name)
		}
		curr = curr.parent
	}

	for i, j := 0, len(components)-1; i < j; i, j = i+1, j-1 {
		components[i], components[j] = components[j], components[i]
	}

	return strings.Join(components, "/")
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
