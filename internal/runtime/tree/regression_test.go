package tree_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// TestParse_RegressionFixtures parses every JSON file under testdata/regression/
// and asserts structural invariants that must hold for any valid schema.
// Add real production schemas to subdirectories (call/, chat/, email/, …).
func TestParse_RegressionFixtures(t *testing.T) {
	const fixturesDir = "testdata/regression"

	if _, err := os.Stat(fixturesDir); os.IsNotExist(err) {
		t.Skip("no regression fixtures — populate testdata/regression/ with real schemas")
	}

	count := 0
	err := filepath.Walk(fixturesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".json" {
			return err
		}
		count++
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			tr, err := tree.ParseJSON(0, raw)
			if err != nil {
				t.Fatalf("ParseJSON: %v", err)
			}
			assertTreeInvariants(t, tr)
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk testdata/regression: %v", err)
	}
	if count == 0 {
		t.Skip("testdata/regression/ exists but contains no .json files")
	}
}

// assertTreeInvariants checks structural properties every parsed tree must hold,
// regardless of schema shape. Failing here means the parser produced a broken
// index — bugs that would cause silent misbehavior at runtime.
func assertTreeInvariants(t *testing.T, tr *tree.Tree) {
	t.Helper()

	if tr.Root == nil {
		t.Fatal("Root is nil")
	}
	if tr.Version == 0 {
		t.Error("Version is 0 — deterministic hash should always be non-zero")
	}

	// Every node reachable from Root must exist in ByID.
	var walk func(*tree.Node)
	walk = func(n *tree.Node) {
		if _, ok := tr.ByID[n.ID]; !ok {
			t.Errorf("node %q reachable from root but absent from ByID", n.ID)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(tr.Root)

	// Every ByTag entry must point to a node that exists in ByID.
	for tag, n := range tr.ByTag {
		if _, ok := tr.ByID[n.ID]; !ok {
			t.Errorf("ByTag[%q] → node %q not in ByID", tag, n.ID)
		}
	}

	// Goto targets that reference missing tags are a schema authoring error,
	// not a parser error — log them as warnings so regressions are visible
	// without causing test failures on schemas that already have dead links.
	var warnGoto func(*tree.Node)
	warnGoto = func(n *tree.Node) {
		if n.OpName == "goto" {
			if target, _ := n.RawArgs.(string); target != "" {
				if _, ok := tr.ByTag[target]; !ok {
					t.Logf("WARN: goto %q at node %q has no matching tag in this schema", target, n.ID)
				}
			}
		}
		for _, c := range n.Children {
			warnGoto(c)
		}
	}
	warnGoto(tr.Root)
}
