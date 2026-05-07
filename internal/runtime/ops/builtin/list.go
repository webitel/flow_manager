package builtin

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// ListDeps is the narrow interface required by the list and listAdd ops.
type ListDeps interface {
	CheckList(domainId int64, number string, listId *int, listName *string) (bool, error)
	AddToList(ctx context.Context, domainId int64, listId *int, listName *string, destination string, description *string, expireAtMS int64) error
}

type listOp struct{ deps ListDeps }
type listAddOp struct{ deps ListDeps }

// ListOp returns the native list op: checks whether destination exists in a
// named/id-addressed list and branches into the actions sub-tree when found.
func ListOp(deps ListDeps) ops.Op { return listOp{deps: deps} }

// ListAddOp returns the native listAdd op: adds a destination number to a list.
func ListAddOp(deps ListDeps) ops.Op { return listAddOp{deps: deps} }

func (listOp) Kind() ops.OpKind    { return ops.OpKindSync }
func (listAddOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o listOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	destination, _ := in.Node.Args["destination"].(string)
	destination = ops.ExpandStr(destination, in.Variables, in.GlobalVar)
	if destination == "" {
		return ops.OpOutput{}, fmt.Errorf("list: destination is required")
	}

	listId, listName := resolveListRef(in.Node.Args)

	exists, err := o.deps.CheckList(in.DomainID, destination, listId, listName)
	if err != nil {
		return ops.OpOutput{}, err
	}

	if exists && len(in.Node.Children) > 0 {
		return ops.OpOutput{Branch: in.Node.Children[0]}, nil
	}
	return ops.OpOutput{}, nil
}

func (o listAddOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	destination, _ := in.Node.Args["destination"].(string)
	destination = ops.ExpandStr(destination, in.Variables, in.GlobalVar)
	if destination == "" {
		return ops.OpOutput{}, fmt.Errorf("listAdd: destination is required")
	}

	listId, listName := resolveListRef(in.Node.Args)

	var description *string
	if d, ok := in.Node.Args["description"].(string); ok && d != "" {
		d = ops.ExpandStr(d, in.Variables, in.GlobalVar)
		description = &d
	}

	var expireAtMS int64
	switch v := in.Node.Args["expireAt"].(type) {
	case float64:
		expireAtMS = int64(v)
	}

	if err := o.deps.AddToList(ctx, in.DomainID, listId, listName, destination, description, expireAtMS); err != nil {
		return ops.OpOutput{}, err
	}
	return ops.OpOutput{}, nil
}

// resolveListRef extracts optional listId and listName from op args.
// Supports both nested {"list": {"id": …, "name": …}} and legacy top-level id/name.
func resolveListRef(args map[string]any) (listId *int, listName *string) {
	if listObj, ok := args["list"].(map[string]any); ok {
		if id, ok := listObj["id"].(float64); ok {
			v := int(id)
			listId = &v
		}
		if name, ok := listObj["name"].(string); ok && name != "" {
			listName = &name
		}
	}
	if listId == nil {
		if id, ok := args["id"].(float64); ok {
			v := int(id)
			listId = &v
		}
	}
	if listName == nil {
		if name, ok := args["name"].(string); ok && name != "" {
			listName = &name
		}
	}
	return
}
