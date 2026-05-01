// Package tree parses a flow schema JSON into a stable, addressable tree of
// nodes. It has no I/O and does not import flow/ or model/.
package tree

// NodeID is a stable identifier assigned at parse time.
//
// For flat sequential nodes the format is a dot-separated path of indices,
// e.g. "0", "1", "2.0.1". When a schema node provides an explicit "_id"
// field that value is used verbatim (and must be unique within the tree).
//
// Container nodes created for composite ops (if-then, if-else, while-do,
// switch-case) receive synthetic IDs such as "0.then", "0.else", "2.do",
// "3.case.active".
type NodeID = string

// Node is one parsed application in the flow schema.
type Node struct {
	ID           NodeID
	OpName       string         // "if", "while", "httpRequest", …
	Args         map[string]any // normalised args used by builtins: map args are flattened; scalars are excluded
	RawArgs      any            // original value from the schema node, used by legacy ops
	Tag          string         // optional goto label
	Break        bool           // stop execution after this node
	Async        bool
	Children     []*Node // sub-trees: then/else for if, do for while, cases for switch
	ParentID     NodeID  // ID of the parent container; empty only for Root
	SiblingIndex int     // index within parent.Children (used by goto)
}

// Tree is the fully parsed, indexed schema.
type Tree struct {
	SchemaID int
	Version  uint64           // deterministic hash of the normalized schema
	Root     *Node            // virtual root; its Children are the top-level apps
	ByID     map[NodeID]*Node // all nodes by ID (including containers)
	ByTag    map[string]*Node // tag label → node (the tagged node itself, not container)
	// Triggers maps trigger names (e.g. "disconnected", "commands-/cancel") to
	// their sub-tree root Node. Populated from the schema's top-level "triggers"
	// element. Empty when schema declares no triggers.
	Triggers map[string]*Node
}
