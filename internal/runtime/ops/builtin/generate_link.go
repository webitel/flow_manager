package builtin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// GenerateLinkDeps is the narrow interface required by the generateLink op.
type GenerateLinkDeps interface {
	GeneratePreSignedLink(ctx context.Context, action, source string, fileId, domainId int64, query map[string]string) (string, error)
}

type generateLinkOp struct{ deps GenerateLinkDeps }

// GenerateLinkOp returns the native generateLink op: produces a pre-signed
// download URL for a file and stores it in a schema variable.
func GenerateLinkOp(deps GenerateLinkDeps) ops.Op { return generateLinkOp{deps: deps} }

func (generateLinkOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o generateLinkOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	setVar, _ := in.Node.Args["set"].(string)
	if setVar == "" {
		return ops.OpOutput{}, fmt.Errorf("generateLink: set is required")
	}

	server, _ := in.Node.Args["server"].(string)
	server = ops.ExpandStr(server, in.Variables, in.GlobalVar)
	server = strings.TrimSuffix(server, "/")

	source, _ := in.Node.Args["source"].(string)
	if source == "" {
		source = "media"
	}

	var fileId int64
	if fileObj, ok := in.Node.Args["file"].(map[string]any); ok {
		if idStr, ok := fileObj["id"].(string); ok && idStr != "" {
			fileId, _ = strconv.ParseInt(idStr, 10, 64)
		}
	}
	if fileId == 0 {
		fileId, _ = strconv.ParseInt(in.ConnID, 10, 64)
	}

	var expire int64
	switch v := in.Node.Args["expire"].(type) {
	case float64:
		expire = int64(v)
	}

	query := make(map[string]string)
	if q, ok := in.Node.Args["query"].(map[string]any); ok {
		for k, v := range q {
			query[k] = fmt.Sprintf("%v", v)
		}
	}
	query["expires"] = strconv.FormatInt(expire*1000, 10)

	link, err := o.deps.GeneratePreSignedLink(ctx, "download", source, fileId, in.DomainID, query)
	if err != nil {
		return ops.OpOutput{}, err
	}

	return ops.OpOutput{SetVars: map[string]string{setVar: server + link}}, nil
}
