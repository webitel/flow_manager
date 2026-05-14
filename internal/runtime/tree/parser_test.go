package tree_test

import (
	"encoding/json"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// helpers

func mustParse(t *testing.T, schemaID int, raw string) *tree.Tree {
	t.Helper()
	tr, err := tree.ParseJSON(schemaID, []byte(raw))
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	return tr
}

func node(t *testing.T, tr *tree.Tree, id string) *tree.Node {
	t.Helper()
	n, ok := tr.ByID[id]
	if !ok {
		t.Fatalf("node %q not found in tree (have %v)", id, nodeIDs(tr))
	}
	return n
}

func nodeIDs(tr *tree.Tree) []string {
	ids := make([]string, 0, len(tr.ByID))
	for id := range tr.ByID {
		ids = append(ids, id)
	}
	return ids
}

// --- tests ---

func TestParse_LinearFlow(t *testing.T) {
	const schema = `[
		{"set": {"foo": "bar"}},
		{"log": {"message": "done"}, "tag": "end"},
		{"set": {"x": "1"}, "break": true}
	]`

	tr := mustParse(t, 1, schema)

	if tr.SchemaID != 1 {
		t.Errorf("SchemaID: got %d", tr.SchemaID)
	}
	if tr.Version == 0 {
		t.Error("Version should be non-zero")
	}
	if len(tr.Root.Children) != 3 {
		t.Fatalf("Root children: got %d, want 3", len(tr.Root.Children))
	}

	n0 := tr.Root.Children[0]
	if n0.ID != "0" || n0.OpName != "set" {
		t.Errorf("node[0]: id=%q op=%q", n0.ID, n0.OpName)
	}

	n1 := tr.Root.Children[1]
	if n1.ID != "1" || n1.OpName != "log" || n1.Tag != "end" {
		t.Errorf("node[1]: id=%q op=%q tag=%q", n1.ID, n1.OpName, n1.Tag)
	}
	if tr.ByTag["end"] != n1 {
		t.Error("ByTag[end] should point to node[1]")
	}

	n2 := tr.Root.Children[2]
	if !n2.Break {
		t.Error("node[2] should have Break=true")
	}
}

func TestParse_IfElse(t *testing.T) {
	const schema = `[
		{"if": {
			"expression": "${x} == 1",
			"then": [{"set": {"result": "yes"}}],
			"else": [{"set": {"result": "no"}}, {"log": {"message": "else"}}]
		}, "tag": "check"}
	]`

	tr := mustParse(t, 2, schema)

	ifNode := node(t, tr, "0")
	if ifNode.OpName != "if" {
		t.Errorf("OpName: got %q", ifNode.OpName)
	}
	if ifNode.Tag != "check" {
		t.Errorf("Tag: got %q", ifNode.Tag)
	}
	// expression preserved in Args, then/else removed
	if _, ok := ifNode.Args["expression"]; !ok {
		t.Error("Args[expression] missing")
	}
	if _, ok := ifNode.Args["then"]; ok {
		t.Error("Args[then] should be removed from Args")
	}
	if _, ok := ifNode.Args["else"]; ok {
		t.Error("Args[else] should be removed from Args")
	}
	if len(ifNode.Children) != 2 {
		t.Fatalf("if Children: got %d, want 2", len(ifNode.Children))
	}

	thenContainer := ifNode.Children[0]
	if thenContainer.ID != "0.then" {
		t.Errorf("then container ID: %q", thenContainer.ID)
	}
	if len(thenContainer.Children) != 1 {
		t.Errorf("then children: %d", len(thenContainer.Children))
	}

	elseContainer := ifNode.Children[1]
	if elseContainer.ID != "0.else" {
		t.Errorf("else container ID: %q", elseContainer.ID)
	}
	if len(elseContainer.Children) != 2 {
		t.Errorf("else children: %d", len(elseContainer.Children))
	}

	// Verify IDs of else children.
	node(t, tr, "0.else.0")
	node(t, tr, "0.else.1")
}

func TestParse_While(t *testing.T) {
	const schema = `[
		{"while": {
			"condition": "${counter} < 5",
			"maxSteps": "100",
			"do": [
				{"set": {"counter": "${counter+1}"}},
				{"log": {"message": "loop"}}
			]
		}}
	]`

	tr := mustParse(t, 3, schema)

	whileNode := node(t, tr, "0")
	if whileNode.OpName != "while" {
		t.Errorf("OpName: got %q", whileNode.OpName)
	}
	if _, ok := whileNode.Args["do"]; ok {
		t.Error("Args[do] should be removed")
	}
	if whileNode.Args["condition"] == nil {
		t.Error("Args[condition] missing")
	}
	if len(whileNode.Children) != 1 {
		t.Fatalf("while Children: got %d, want 1", len(whileNode.Children))
	}

	doContainer := whileNode.Children[0]
	if doContainer.ID != "0.do" {
		t.Errorf("do container ID: %q", doContainer.ID)
	}
	if len(doContainer.Children) != 2 {
		t.Errorf("do children: %d", len(doContainer.Children))
	}
	node(t, tr, "0.do.0")
	node(t, tr, "0.do.1")
}

func TestParse_Switch(t *testing.T) {
	const schema = `[
		{"switch": {
			"variable": "${status}",
			"case": {
				"active":  [{"set": {"msg": "on"}}],
				"default": [{"set": {"msg": "off"}}, {"log": {"message": "x"}}]
			}
		}}
	]`

	tr := mustParse(t, 4, schema)

	sw := node(t, tr, "0")
	if sw.OpName != "switch" {
		t.Errorf("OpName: got %q", sw.OpName)
	}
	if _, ok := sw.Args["case"]; ok {
		t.Error("Args[case] should be removed")
	}

	idx, ok := sw.Args["_cases_index"].(map[string]int)
	if !ok {
		t.Fatalf("_cases_index missing or wrong type: %T", sw.Args["_cases_index"])
	}
	if idx["active"] < 0 || idx["default"] < 0 {
		t.Errorf("_cases_index: %v", idx)
	}
	if len(sw.Children) != 2 {
		t.Fatalf("switch Children: %d, want 2", len(sw.Children))
	}

	// active comes before default alphabetically → index 0
	if idx["active"] != 0 {
		t.Errorf("active index: got %d, want 0", idx["active"])
	}
	if idx["default"] != 1 {
		t.Errorf("default index: got %d, want 1", idx["default"])
	}

	// Verify container IDs and their children.
	node(t, tr, "0.case.active")
	node(t, tr, "0.case.default")
	node(t, tr, "0.case.active.0")
	node(t, tr, "0.case.default.0")
	node(t, tr, "0.case.default.1")
}

func TestParse_ExplicitID(t *testing.T) {
	const schema = `[
		{"set": {"x": "1"}, "_id": "my-node"}
	]`

	tr := mustParse(t, 5, schema)

	n := node(t, tr, "my-node")
	if n.OpName != "set" {
		t.Errorf("OpName: %q", n.OpName)
	}
	if tr.Root.Children[0].ID != "my-node" {
		t.Errorf("Root child ID: %q", tr.Root.Children[0].ID)
	}
}

func TestParse_NestedIfInsideWhile(t *testing.T) {
	const schema = `[
		{"while": {
			"condition": "${ok}",
			"do": [
				{"if": {
					"expression": "${x} > 0",
					"then": [{"set": {"y": "1"}}]
				}}
			]
		}}
	]`

	tr := mustParse(t, 6, schema)

	node(t, tr, "0")             // while
	node(t, tr, "0.do")          // while container
	node(t, tr, "0.do.0")        // if inside while
	node(t, tr, "0.do.0.then")   // if-then container
	node(t, tr, "0.do.0.else")   // if-else container (empty but registered)
	node(t, tr, "0.do.0.then.0") // set inside if-then
}

func TestHashSchema_Deterministic(t *testing.T) {
	const schema = `[
		{"set": {"a": "1"}},
		{"if": {"expression": "true", "then": [{"log": {"message": "x"}}]}}
	]`

	tr1 := mustParse(t, 7, schema)
	tr2 := mustParse(t, 7, schema)

	if tr1.Version != tr2.Version {
		t.Errorf("version not deterministic: %d vs %d", tr1.Version, tr2.Version)
	}
}

func TestHashSchema_DifferentSchemaDifferentHash(t *testing.T) {
	const s1 = `[{"set": {"a": "1"}}]`
	const s2 = `[{"set": {"a": "2"}}]`

	tr1 := mustParse(t, 8, s1)
	tr2 := mustParse(t, 8, s2)

	if tr1.Version == tr2.Version {
		t.Error("different schemas should produce different versions")
	}
}

func TestParse_BreakFlag(t *testing.T) {
	const schema = `[{"break": true}]`

	tr := mustParse(t, 9, schema)

	if len(tr.Root.Children) != 1 {
		t.Fatalf("children: %d", len(tr.Root.Children))
	}
	n := tr.Root.Children[0]
	if n.OpName != "break" {
		t.Errorf("OpName: %q", n.OpName)
	}
	if !n.Break {
		t.Error("Break flag should be set")
	}
}

func TestParse_EmptySchema(t *testing.T) {
	tr := mustParse(t, 10, `[]`)

	if len(tr.Root.Children) != 0 {
		t.Errorf("expected no children, got %d", len(tr.Root.Children))
	}
	if tr.Version == 0 {
		t.Error("Version should be non-zero even for empty schema")
	}
}

func TestParse_ScalarArgs(t *testing.T) {
	// Some ops take a plain value, not a map.
	const schema = `[{"sleep": 3000}]`

	tr := mustParse(t, 11, schema)

	n := node(t, tr, "0")
	if n.OpName != "sleep" {
		t.Errorf("OpName: %q", n.OpName)
	}
	// Scalar stored under the op name key.
	if n.Args["sleep"] == nil {
		t.Error("Args[sleep] should hold the scalar value")
	}
}

func TestParse_AllNodeIDsInByID(t *testing.T) {
	const schema = `[
		{"set": {"a": "1"}},
		{"if": {
			"expression": "true",
			"then": [{"log": {"message": "t"}}],
			"else": [{"log": {"message": "e"}}]
		}},
		{"while": {"condition": "false", "do": [{"set": {"b": "2"}}]}}
	]`

	tr := mustParse(t, 12, schema)

	// Verify every node reachable from root is in ByID.
	var walk func(*tree.Node)
	walk = func(n *tree.Node) {
		if _, ok := tr.ByID[n.ID]; !ok {
			t.Errorf("node %q not in ByID", n.ID)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(tr.Root)
}

func TestParse_Triggers(t *testing.T) {
	const schema = `[
		{
			"triggers": {
				"disconnected": [{"log": {"message": "bye"}}],
				"commands": {
					"/cancel": [{"log": {"message": "cancelled"}}],
					"/help":   [{"log": {"message": "help"}}]
				}
			}
		},
		{"recvMessage": {"set": "answer"}}
	]`

	tr := mustParse(t, 14, schema)

	// triggers element must NOT appear in root children.
	if len(tr.Root.Children) != 1 {
		t.Fatalf("Root.Children: got %d, want 1 (only recvMessage)", len(tr.Root.Children))
	}
	if tr.Root.Children[0].OpName != "recvMessage" {
		t.Errorf("root child OpName: %q", tr.Root.Children[0].OpName)
	}

	// Flat trigger.
	disc, ok := tr.Triggers["disconnected"]
	if !ok {
		t.Fatal("Triggers[disconnected] missing")
	}
	if len(disc.Children) != 1 || disc.Children[0].OpName != "log" {
		t.Errorf("disconnected trigger children: %v", disc.Children)
	}
	if disc.ID != "trigger.disconnected" {
		t.Errorf("disconnected container ID: %q", disc.ID)
	}

	// Command triggers.
	for _, cmd := range []string{"/cancel", "/help"} {
		key := "commands-" + cmd
		trig, ok := tr.Triggers[key]
		if !ok {
			t.Errorf("Triggers[%q] missing", key)
			continue
		}
		if len(trig.Children) != 1 {
			t.Errorf("Triggers[%q] children: %d", key, len(trig.Children))
		}
	}
}

func TestParse_TriggersOnly(t *testing.T) {
	// Schema with only a triggers element and no regular ops — should parse cleanly.
	const schema = `[{"triggers": {"disconnected": [{"log": {"message": "x"}}]}}]`

	tr := mustParse(t, 15, schema)

	if len(tr.Root.Children) != 0 {
		t.Errorf("Root.Children: got %d, want 0", len(tr.Root.Children))
	}
	if _, ok := tr.Triggers["disconnected"]; !ok {
		t.Error("Triggers[disconnected] missing")
	}
}

func TestParse_LegacyTriggerSingular(t *testing.T) {
	// Legacy schemas use {"trigger": {...}} (singular). The parser must treat it
	// identically to {"triggers": {...}} (plural).
	const schema = `[
		{"trigger": {
			"disconnected": [{"log": {"message": "bye"}}],
			"commands": {
				"/cancel": [{"log": {"message": "cancelled"}}]
			}
		}},
		{"sendText": "hello"}
	]`

	tr := mustParse(t, 16, schema)

	// Trigger element must NOT appear as a child node.
	if len(tr.Root.Children) != 1 {
		t.Fatalf("Root.Children: got %d, want 1", len(tr.Root.Children))
	}
	if tr.Root.Children[0].OpName != "sendText" {
		t.Errorf("root child: %q", tr.Root.Children[0].OpName)
	}

	if _, ok := tr.Triggers["disconnected"]; !ok {
		t.Error("Triggers[disconnected] missing")
	}
	if _, ok := tr.Triggers["commands-/cancel"]; !ok {
		t.Error("Triggers[commands-/cancel] missing")
	}
}

func TestParse_Goto(t *testing.T) {
	const schema = `[
		{"log": {"message": "step1"}, "tag": "start"},
		{"goto": "start"}
	]`

	tr := mustParse(t, 20, schema)

	gotoNode := node(t, tr, "1")
	if gotoNode.OpName != "goto" {
		t.Fatalf("OpName: %q", gotoNode.OpName)
	}
	// goto target is stored as scalar under its own op key.
	target, _ := gotoNode.Args["goto"].(string)
	if target != "start" {
		t.Errorf("goto target: got %q, want %q", target, "start")
	}
}

func TestParse_FunctionDef(t *testing.T) {
	const schema = `[
		{"function": {"name": "greet", "actions": [
			{"log": {"message": "hello"}},
			{"set": {"greeted": "true"}}
		]}},
		{"execute": {"name": "greet"}}
	]`

	tr := mustParse(t, 21, schema)

	// function node must NOT appear in Root.Children.
	if len(tr.Root.Children) != 1 {
		t.Fatalf("Root.Children: got %d, want 1 (only execute)", len(tr.Root.Children))
	}
	if tr.Root.Children[0].OpName != "execute" {
		t.Errorf("root child OpName: %q", tr.Root.Children[0].OpName)
	}

	// function body must be indexed in Tree.Functions.
	fn, ok := tr.Functions["greet"]
	if !ok {
		t.Fatal("Functions[greet] missing")
	}
	if fn.ID != "function.greet" {
		t.Errorf("function container ID: %q", fn.ID)
	}
	if len(fn.Children) != 2 {
		t.Errorf("function body children: got %d, want 2", len(fn.Children))
	}
	if fn.Children[0].OpName != "log" {
		t.Errorf("fn body[0]: %q", fn.Children[0].OpName)
	}
	if fn.Children[1].OpName != "set" {
		t.Errorf("fn body[1]: %q", fn.Children[1].OpName)
	}

	// function body nodes must be in ByID.
	node(t, tr, "function.greet.0")
	node(t, tr, "function.greet.1")

	// execute node carries the function name in Args.
	execNode := tr.Root.Children[0]
	name, _ := execNode.Args["name"].(string)
	if name != "greet" {
		t.Errorf("execute Args[name]: got %q, want %q", name, "greet")
	}
}

func TestParse_FunctionMissingName(t *testing.T) {
	const schema = `[{"function": {"actions": [{"log": {"message": "x"}}]}}]`
	_, err := tree.ParseJSON(22, []byte(schema))
	if err == nil {
		t.Fatal("expected error for function without name")
	}
}

// ── list ──────────────────────────────────────────────────────────────────────

func TestParse_List(t *testing.T) {
	const schema = `[
		{"list": {
			"data":    "${myList}",
			"itemVar": "item",
			"actions": [
				{"log": {"message": "processing"}},
				{"set": {"done": "true"}}
			]
		}}
	]`

	tr := mustParse(t, 30, schema)

	listNode := node(t, tr, "0")
	if listNode.OpName != "list" {
		t.Fatalf("OpName: %q", listNode.OpName)
	}
	// "actions" must be moved to a child container.
	if _, ok := listNode.Args["actions"]; ok {
		t.Error("Args[actions] should be removed from list node")
	}
	if len(listNode.Children) != 1 {
		t.Fatalf("list Children: got %d, want 1", len(listNode.Children))
	}

	actionsContainer := listNode.Children[0]
	if actionsContainer.ID != "0.actions" {
		t.Errorf("actions container ID: %q", actionsContainer.ID)
	}
	if len(actionsContainer.Children) != 2 {
		t.Errorf("actions children: got %d, want 2", len(actionsContainer.Children))
	}
	node(t, tr, "0.actions.0")
	node(t, tr, "0.actions.1")
}

func TestParse_List_NoActions(t *testing.T) {
	const schema = `[{"list": {"data": "${xs}", "itemVar": "x"}}]`

	tr := mustParse(t, 31, schema)

	listNode := node(t, tr, "0")
	if len(listNode.Children) != 0 {
		t.Errorf("list without actions should have 0 children, got %d", len(listNode.Children))
	}
}

// ── formTable ─────────────────────────────────────────────────────────────────

func TestParse_FormTable(t *testing.T) {
	const schema = `[
		{"formTable": {
			"formId": "123",
			"outputs": {
				"reject":  [{"set": {"status": "rejected"}}],
				"approve": [{"set": {"status": "approved"}}, {"log": {"message": "ok"}}]
			}
		}}
	]`

	tr := mustParse(t, 32, schema)

	ft := node(t, tr, "0")
	if ft.OpName != "formTable" {
		t.Fatalf("OpName: %q", ft.OpName)
	}
	if _, ok := ft.Args["outputs"]; ok {
		t.Error("Args[outputs] should be removed")
	}

	idx, ok := ft.Args["_outputs_index"].(map[string]int)
	if !ok {
		t.Fatalf("_outputs_index missing or wrong type: %T", ft.Args["_outputs_index"])
	}
	// Sorted alphabetically: approve(0), reject(1).
	if idx["approve"] != 0 {
		t.Errorf("approve index: got %d, want 0", idx["approve"])
	}
	if idx["reject"] != 1 {
		t.Errorf("reject index: got %d, want 1", idx["reject"])
	}
	if len(ft.Children) != 2 {
		t.Fatalf("formTable Children: got %d, want 2", len(ft.Children))
	}

	node(t, tr, "0.output.approve")
	node(t, tr, "0.output.reject")
	node(t, tr, "0.output.approve.0") // set inside approve
	node(t, tr, "0.output.reject.0")  // set inside reject
}

func TestParse_FormTable_NoOutputs(t *testing.T) {
	const schema = `[{"formTable": {"formId": "abc"}}]`

	tr := mustParse(t, 33, schema)

	ft := node(t, tr, "0")
	if len(ft.Children) != 0 {
		t.Errorf("formTable without outputs: got %d children, want 0", len(ft.Children))
	}
}

// ── bridge / joinAgent hooks ──────────────────────────────────────────────────

func TestParse_Bridge_Hooks(t *testing.T) {
	const schema = `[
		{"bridge": {
			"destination": "sip:agent@domain",
			"timeout":     30,
			"bridged": [
				{"log": {"message": "call connected"}},
				{"set": {"result": "bridged"}}
			]
		}}
	]`

	tr := mustParse(t, 34, schema)

	bridgeNode := node(t, tr, "0")
	if bridgeNode.OpName != "bridge" {
		t.Fatalf("OpName: %q", bridgeNode.OpName)
	}
	if _, ok := bridgeNode.Args["bridged"]; ok {
		t.Error("Args[bridged] should be removed")
	}

	hooksIdx, ok := bridgeNode.Args["_hooks_index"].(map[string]int)
	if !ok {
		t.Fatalf("_hooks_index missing: %T", bridgeNode.Args["_hooks_index"])
	}
	if hooksIdx["bridged"] != 0 {
		t.Errorf("bridged hook index: got %d, want 0", hooksIdx["bridged"])
	}
	if len(bridgeNode.Children) != 1 {
		t.Fatalf("bridge Children: got %d, want 1", len(bridgeNode.Children))
	}

	bridgedContainer := bridgeNode.Children[0]
	if bridgedContainer.ID != "0.bridged" {
		t.Errorf("bridged container ID: %q", bridgedContainer.ID)
	}
	if len(bridgedContainer.Children) != 2 {
		t.Errorf("bridged children: got %d, want 2", len(bridgedContainer.Children))
	}
}

func TestParse_Bridge_EmptyHookSkipped(t *testing.T) {
	// Empty bridged array must not create a child container.
	const schema = `[{"bridge": {"destination": "sip:x", "bridged": []}}]`

	tr := mustParse(t, 35, schema)

	bridgeNode := node(t, tr, "0")
	if len(bridgeNode.Children) != 0 {
		t.Errorf("empty bridged should produce 0 children, got %d", len(bridgeNode.Children))
	}
	if _, ok := bridgeNode.Args["_hooks_index"]; ok {
		t.Error("_hooks_index must not be set when all hooks are empty")
	}
}

func TestParse_JoinAgent_Hooks(t *testing.T) {
	const schema = `[
		{"joinAgent": {
			"agentId": "42",
			"bridged": [{"set": {"transfer": "true"}}]
		}}
	]`

	tr := mustParse(t, 36, schema)

	ja := node(t, tr, "0")
	if ja.OpName != "joinAgent" {
		t.Fatalf("OpName: %q", ja.OpName)
	}
	if _, ok := ja.Args["bridged"]; ok {
		t.Error("Args[bridged] should be removed")
	}
	hooksIdx, ok := ja.Args["_hooks_index"].(map[string]int)
	if !ok {
		t.Fatalf("_hooks_index missing: %T", ja.Args["_hooks_index"])
	}
	if hooksIdx["bridged"] != 0 {
		t.Errorf("bridged index: got %d, want 0", hooksIdx["bridged"])
	}
}

// ── joinQueue timers ──────────────────────────────────────────────────────────

func TestParse_JoinQueue_Timers(t *testing.T) {
	const schema = `[
		{"joinQueue": {
			"queueId": "99",
			"timers": [
				{
					"condition": "${x} > 0",
					"actions": [
						{"log": {"message": "timer fired"}},
						{"set": {"timed_out": "true"}}
					]
				},
				{
					"condition": "false"
				}
			]
		}}
	]`

	tr := mustParse(t, 37, schema)

	jq := node(t, tr, "0")
	if jq.OpName != "joinQueue" {
		t.Fatalf("OpName: %q", jq.OpName)
	}

	timers, ok := jq.Args["timers"].([]any)
	if !ok {
		t.Fatalf("Args[timers] wrong type: %T", jq.Args["timers"])
	}

	// Timer[0]: has actions → container created, actions deleted, _children_idx set.
	t0, _ := timers[0].(map[string]any)
	if _, still := t0["actions"]; still {
		t.Error("timers[0].actions should be removed after parsing")
	}
	childIdx, _ := t0["_children_idx"].(int)
	if childIdx != 0 {
		t.Errorf("timers[0]._children_idx: got %d, want 0", childIdx)
	}

	// Timer[1]: no actions → not touched, no _children_idx.
	t1, _ := timers[1].(map[string]any)
	if _, has := t1["_children_idx"]; has {
		t.Error("timers[1] should not have _children_idx (no actions)")
	}

	// Container registered in ByID and accessible as child.
	node(t, tr, "0.timer.0")
	node(t, tr, "0.timer.0.0") // log inside timer
	node(t, tr, "0.timer.0.1") // set inside timer

	// Timer[1] (no actions) must NOT produce a container.
	if _, exists := tr.ByID["0.timer.1"]; exists {
		t.Error("timer without actions must not create a container node")
	}
}

func TestParse_JoinQueue_TimersAndHooks(t *testing.T) {
	// Verifies that timers and hooks both contribute to Children and that
	// _hooks_index accounts for the timer children that precede the hooks.
	const schema = `[
		{"joinQueue": {
			"queueId": "5",
			"timers": [
				{"condition": "true", "actions": [{"log": {"message": "t"}}]}
			],
			"bridged": [{"set": {"call": "done"}}],
			"waiting": [{"log": {"message": "waiting"}}]
		}}
	]`

	tr := mustParse(t, 38, schema)

	jq := node(t, tr, "0")

	// 1 timer child + 2 hook children = 3 total.
	if len(jq.Children) != 3 {
		t.Fatalf("Children: got %d, want 3 (1 timer + bridged + waiting)", len(jq.Children))
	}

	hooksIdx, ok := jq.Args["_hooks_index"].(map[string]int)
	if !ok {
		t.Fatalf("_hooks_index missing")
	}
	// Timer occupies index 0; hooks follow in order: waiting, offering, bridged, reporting.
	// waiting → index 1 (len=1 before append), bridged → index 2 (len=2 before append).
	if hooksIdx["waiting"] != 1 {
		t.Errorf("waiting index: got %d, want 1", hooksIdx["waiting"])
	}
	if hooksIdx["bridged"] != 2 {
		t.Errorf("bridged index: got %d, want 2", hooksIdx["bridged"])
	}
}

// ── error paths ───────────────────────────────────────────────────────────────

func TestParseJSON_InvalidJSON(t *testing.T) {
	_, err := tree.ParseJSON(99, []byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParse_TriggersBadSchema(t *testing.T) {
	// triggers value must be an object, not an array.
	const schema = `[{"triggers": ["bad"]}]`
	_, err := tree.ParseJSON(100, []byte(schema))
	if err == nil {
		t.Fatal("expected error for non-object triggers value")
	}
}

func TestParse_SwitchCaseBadSchema(t *testing.T) {
	// switch.case must be an object (map), not an array.
	const schema = `[{"switch": {"variable": "${x}", "case": ["bad"]}}]`
	_, err := tree.ParseJSON(101, []byte(schema))
	if err == nil {
		t.Fatal("expected error for non-object switch.case")
	}
}

func TestParse_IfBadThenSchema(t *testing.T) {
	// if.then must be an array.
	const schema = `[{"if": {"expression": "true", "then": "not-an-array"}}]`
	_, err := tree.ParseJSON(102, []byte(schema))
	if err == nil {
		t.Fatal("expected error for non-array if.then")
	}
}

func TestParse_WhileBadDoSchema(t *testing.T) {
	const schema = `[{"while": {"condition": "true", "do": "not-an-array"}}]`
	_, err := tree.ParseJSON(103, []byte(schema))
	if err == nil {
		t.Fatal("expected error for non-array while.do")
	}
}

func TestParse_JSONRoundTripOfArgs(t *testing.T) {
	const schema = `[{"httpRequest": {"url": "https://example.com", "method": "GET", "timeout": 5000}}]`

	tr := mustParse(t, 13, schema)

	n := node(t, tr, "0")
	b, err := json.Marshal(n.Args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal args: %v", err)
	}
	if got["url"] != "https://example.com" {
		t.Errorf("url: %v", got["url"])
	}
}
