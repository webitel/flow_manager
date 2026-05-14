package testutil

import (
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// MakeInput builds an OpInput for unit tests.
// args maps to Node.Args; vars is the current variable snapshot.
func MakeInput(args map[string]any, vars map[string]string) ops.OpInput {
	return ops.OpInput{
		Node:      &tree.Node{Args: args},
		Variables: vars,
	}
}

// MakeInputWithDomain builds an OpInput that carries a domain ID (needed by
// cache, sql, and similar domain-scoped ops).
func MakeInputWithDomain(domainID int64, args map[string]any, vars map[string]string) ops.OpInput {
	return ops.OpInput{
		Node:      &tree.Node{Args: args},
		DomainID:  domainID,
		Variables: vars,
	}
}
