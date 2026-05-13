package tree_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/tree"
)

var update = flag.Bool("update", false, "regenerate .snap files instead of comparing")

// TestParse_RegressionFixtures parses every *.json under testdata/regression/
// and compares the resulting tree structure against a golden *.snap file.
//
// First run (or after intentional parser change):
//
//	go test ./internal/runtime/tree/... -run Regression -update
//
// Normal CI run (no flag): fails if the parsed tree diverges from the snapshot.
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
			checkSnapshot(t, path, tr)
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if count == 0 {
		t.Skip("testdata/regression/ exists but no .json files found")
	}
}

// checkSnapshot compares tr against the golden .snap file next to the fixture.
// With -update it writes a new snapshot instead of comparing.
func checkSnapshot(t *testing.T, fixturePath string, tr *tree.Tree) {
	t.Helper()

	snapPath := strings.TrimSuffix(fixturePath, ".json") + ".snap"
	got := snapshotTree(tr)

	if *update {
		if err := os.WriteFile(snapPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write snapshot %s: %v", snapPath, err)
		}
		t.Logf("snapshot updated: %s", snapPath)
		return
	}

	wantBytes, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatalf("snapshot %s missing — run with -update to create it", snapPath)
	}
	want := string(wantBytes)

	if got == want {
		return
	}

	// Print a line-level diff so failures are readable.
	wantLines := strings.Split(strings.TrimRight(want, "\n"), "\n")
	gotLines := strings.Split(strings.TrimRight(got, "\n"), "\n")

	var sb strings.Builder
	max := len(wantLines)
	if len(gotLines) > max {
		max = len(gotLines)
	}
	for i := 0; i < max; i++ {
		w, g := "", ""
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			fmt.Fprintf(&sb, "line %d\n  snap: %s\n   got: %s\n", i+1, w, g)
		}
	}
	t.Fatalf("tree structure differs from snapshot %s:\n%s\nrun with -update to accept new output",
		snapPath, sb.String())
}

// snapshotTree serialises the parsed tree as a human-readable text that can be
// stored in git and diffed easily. Format (tab-separated columns):
//
//	{id}  {op}  {tag}  {childCount}
//
// Containers (synthetic then/else/do/case nodes) have op "<seq>".
// The virtual root is omitted. Triggers follow after "=== triggers ===".
func snapshotTree(tr *tree.Tree) string {
	var sb strings.Builder

	var walk func(n *tree.Node)
	walk = func(n *tree.Node) {
		op := n.OpName
		if op == "" {
			op = "<seq>"
		}
		tag := n.Tag
		fmt.Fprintf(&sb, "%s\t%s\t%s\t%d\n", n.ID, op, tag, len(n.Children))
		for _, c := range n.Children {
			walk(c)
		}
	}
	for _, c := range tr.Root.Children {
		walk(c)
	}

	if len(tr.Triggers) > 0 {
		sb.WriteString("=== triggers ===\n")
		names := make([]string, 0, len(tr.Triggers))
		for name := range tr.Triggers {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			n := tr.Triggers[name]
			fmt.Fprintf(&sb, "%s\t%d\n", name, len(n.Children))
		}
	}

	return sb.String()
}

// assertTreeInvariants checks structural properties that must hold for any
// parsed tree, independent of schema content.
func assertTreeInvariants(t *testing.T, tr *tree.Tree) {
	t.Helper()

	if tr.Root == nil {
		t.Fatal("Root is nil")
	}
	if tr.Version == 0 {
		t.Error("Version is 0 — should always be non-zero")
	}

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

	for tag, n := range tr.ByTag {
		if _, ok := tr.ByID[n.ID]; !ok {
			t.Errorf("ByTag[%q] → node %q not in ByID", tag, n.ID)
		}
	}
}
