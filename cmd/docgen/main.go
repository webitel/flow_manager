// Command docgen generates bin/doc.yaml from op Doc() implementations.
// Run via: go generate ./internal/runtime/ops/
package main

import (
	"os"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/ops/domain/calendar"
)

func main() {
	reg := ops.NewRegistry()
	builtin.Register(reg)
	reg.Register("calendar", calendar.New(nil))

	all := reg.All()

	var names []string
	for name, op := range all {
		if _, ok := op.(ops.Documenter); ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	docs := make(map[string]ops.OpDoc, len(names))
	for _, name := range names {
		docs[name] = all[name].(ops.Documenter).Doc()
	}

	data, err := yaml.Marshal(docs)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("doc.yaml", data, 0o644); err != nil {
		panic(err)
	}
}
