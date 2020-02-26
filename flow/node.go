package flow

type BaseNode struct {
	_depth int16
	idx    int
	parent *Node
}

type Node struct {
	BaseNode
	position int
	children []*ApplicationRequest
}

func (n *BaseNode) getDepth() int16 {
	return n._depth
}

func (n *BaseNode) setParentNode(parent *Node) {
	if parent != nil {
		n.parent = parent
		n.idx = parent.getLength()
		n._depth = parent._depth + 1
	}
}

func (n *BaseNode) GetParent() *Node {
	if n.parent != nil {
		return n.parent
	}
	return nil
}

func (n *Node) getLength() int {
	return len(n.children)
}

func (n *Node) Add(newNode ApplicationRequest) {
	n.children = append(n.children, &newNode)
}

func (n *Node) setFirst() {
	n.position = 0
}

func (n *Node) Next() (newNode *ApplicationRequest) {
	if n.position < n.getLength() {
		newNode = n.children[n.position]
		n.position++
	}
	return
}

func NewNode(parent *Node) *Node {
	n := &Node{}
	n.setParentNode(parent)

	return n
}
