package filetree

type Node struct {
	Name string
	parent *Node
	IsDir bool
	children []*Node
}

func NewNode(name string, isdir bool, parent *Node) *Node {
	return &Node{
		Name: name,
		parent: parent,
		IsDir: isdir,
	}
}

func (n *Node) AddChild(child *Node) {
	n.children = append(n.children, child)
	child.parent = n
}
