package tree

import (
	"encoding/json"
	"fmt"
)

// metaKeys are the ApplicationObject fields that are NOT the op name/args.
var metaKeys = map[string]bool{
	"_id":   true,
	"break": true,
	"async": true,
	"tag":   true,
	"limit": true,
	"trace": true,
}

// Schema is the raw schema as it comes from the database ([]ApplicationObject).
// Using []map[string]any avoids importing model/ or flow/.
type Schema = []map[string]any

// Parse converts a raw schema and schema ID into an indexed Tree.
// The schema must already be unmarshalled from JSON (i.e. json.Unmarshal into
// []map[string]any). Use ParseJSON if you have raw bytes.
func Parse(schemaID int, schema Schema) (*Tree, error) {
	t := &Tree{
		SchemaID: schemaID,
		ByID:     make(map[NodeID]*Node),
		ByTag:    make(map[string]*Node),
	}

	root := &Node{ID: "root", OpName: ""}
	t.Root = root
	t.ByID["root"] = root

	if err := parseApps(t, root, schema, ""); err != nil {
		return nil, err
	}

	t.Version = hashSchema(schema)
	return t, nil
}

// ParseJSON is a convenience wrapper that unmarshals raw JSON bytes before
// calling Parse.
func ParseJSON(schemaID int, data []byte) (*Tree, error) {
	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("tree: unmarshal schema: %w", err)
	}
	return Parse(schemaID, schema)
}

// parseApps appends parsed nodes to parent.Children for every element in apps.
// prefix is the dot-separated path prefix for stable NodeID assignment.
func parseApps(t *Tree, parent *Node, apps Schema, prefix string) error {
	for i, obj := range apps {
		var idPrefix string
		if prefix == "" {
			idPrefix = fmt.Sprintf("%d", i)
		} else {
			idPrefix = fmt.Sprintf("%s.%d", prefix, i)
		}

		node, err := parseNode(t, obj, idPrefix)
		if err != nil {
			return err
		}
		if node == nil {
			continue
		}

		node.ParentID = parent.ID
		node.SiblingIndex = len(parent.Children)
		parent.Children = append(parent.Children, node)
		t.ByID[node.ID] = node
		if node.Tag != "" {
			t.ByTag[node.Tag] = node
		}
	}
	return nil
}

// parseNode converts one ApplicationObject into a Node.
// Returns nil if the object has no recognisable op name (skip silently, same
// as the legacy parser behaviour).
func parseNode(t *Tree, obj map[string]any, id string) (*Node, error) {
	node := &Node{
		Args: make(map[string]any),
	}

	// Resolve explicit _id override.
	if v, ok := obj["_id"].(string); ok && v != "" {
		id = v
	}
	node.ID = id

	// Parse meta fields.
	if v, ok := obj["break"].(bool); ok {
		node.Break = v
	}
	if v, ok := obj["async"].(bool); ok {
		node.Async = v
	}
	if v, ok := obj["tag"].(string); ok {
		node.Tag = v
	}

	// Find the op name: the first non-meta key.
	for k, v := range obj {
		if metaKeys[k] {
			continue
		}
		node.OpName = k

		// Normalise args to map[string]any.
		switch val := v.(type) {
		case map[string]any:
			for argK, argV := range val {
				node.Args[argK] = argV
			}
		default:
			// Scalar or array — store under the op name key so the op can read it.
			node.Args[k] = v
		}
		break // only the first non-meta key is the op
	}

	// "break" with no op name is a synthetic break op.
	if node.OpName == "" && node.Break {
		node.OpName = "break"
	}

	if node.OpName == "" {
		return nil, nil
	}

	// Parse composite ops — extract nested app arrays into Children and remove
	// them from Args so Args stays flat.
	switch node.OpName {
	case "if":
		if err := parseIfChildren(t, node, id); err != nil {
			return nil, err
		}
	case "while":
		if err := parseWhileChildren(t, node, id); err != nil {
			return nil, err
		}
	case "switch":
		if err := parseSwitchChildren(t, node, id); err != nil {
			return nil, err
		}
	}

	return node, nil
}

func parseIfChildren(t *Tree, node *Node, id string) error {
	// then branch
	if raw, ok := node.Args["then"]; ok {
		delete(node.Args, "then")
		apps, err := toSchema(raw)
		if err != nil {
			return fmt.Errorf("tree: if.then at %s: %w", id, err)
		}
		container := newContainer(id+".then", t)
		if err := parseApps(t, container, apps, id+".then"); err != nil {
			return err
		}
		node.Children = append(node.Children, container)
	} else {
		// Ensure index 0 always exists for then, even if empty.
		container := newContainer(id+".then", t)
		node.Children = append(node.Children, container)
	}

	// else branch
	if raw, ok := node.Args["else"]; ok {
		delete(node.Args, "else")
		apps, err := toSchema(raw)
		if err != nil {
			return fmt.Errorf("tree: if.else at %s: %w", id, err)
		}
		container := newContainer(id+".else", t)
		if err := parseApps(t, container, apps, id+".else"); err != nil {
			return err
		}
		node.Children = append(node.Children, container)
	} else {
		container := newContainer(id+".else", t)
		node.Children = append(node.Children, container)
	}

	return nil
}

func parseWhileChildren(t *Tree, node *Node, id string) error {
	raw, ok := node.Args["do"]
	if !ok {
		return nil
	}
	delete(node.Args, "do")

	apps, err := toSchema(raw)
	if err != nil {
		return fmt.Errorf("tree: while.do at %s: %w", id, err)
	}
	container := newContainer(id+".do", t)
	if err := parseApps(t, container, apps, id+".do"); err != nil {
		return err
	}
	node.Children = append(node.Children, container)
	return nil
}

func parseSwitchChildren(t *Tree, node *Node, id string) error {
	raw, ok := node.Args["case"]
	if !ok {
		return nil
	}
	delete(node.Args, "case")

	cases, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("tree: switch.case at %s must be an object", id)
	}

	// Deterministic ordering: sort case names so NodeIDs are stable.
	caseIndex := make(map[string]int, len(cases))
	names := sortedKeys(cases)
	for i, name := range names {
		caseID := id + ".case." + name
		container := newContainer(caseID, t)
		apps, err := toSchema(cases[name])
		if err != nil {
			return fmt.Errorf("tree: switch.case[%s] at %s: %w", name, id, err)
		}
		if err := parseApps(t, container, apps, caseID); err != nil {
			return err
		}
		node.Children = append(node.Children, container)
		caseIndex[name] = i
	}

	// Store the index so the switch op can resolve case name → children index.
	node.Args["_cases_index"] = caseIndex
	return nil
}

// newContainer creates and registers an unnamed container node used to group
// branches of composite ops (if/while/switch).
func newContainer(id string, t *Tree) *Node {
	c := &Node{ID: id, OpName: ""}
	t.ByID[id] = c
	return c
}

// toSchema coerces a raw value (expected to be []interface{} from JSON
// unmarshal) into Schema ([]map[string]any).
func toSchema(raw any) (Schema, error) {
	if raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", raw)
	}
	out := make(Schema, 0, len(arr))
	for _, el := range arr {
		m, ok := el.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected object element, got %T", el)
		}
		out = append(out, m)
	}
	return out, nil
}

// sortedKeys returns the keys of m in sorted order for deterministic iteration.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort — case maps are small.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
