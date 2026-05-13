package email

import (
	"context"
	"fmt"
	"regexp"

	"github.com/webitel/flow_manager/gen/contacts"
	domaincontacts "github.com/webitel/flow_manager/internal/domain/contacts"
	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	emailop "github.com/webitel/flow_manager/internal/runtime/ops/domain/email"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/runtime/sessionmgr"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/model"
)

// emailChannel is the channel discriminator stored in flow.runtime_state.
// Matches model.ConnectionTypeEmail (iota = 2).
const emailChannel = int16(model.ConnectionTypeEmail)

var compileVar = regexp.MustCompile(`\$\{([\s\S]*?)\}`)

type Router struct {
	fm         Deps
	contacts   domaincontacts.Client
	driver     *interpreter.Driver
	sessionMgr *sessionmgr.Manager
}

func Init(deps Deps, contacts domaincontacts.Client) model.Router {
	r := &Router{fm: deps, contacts: contacts}

	kit := runtimekit.Bootstrap(runtimekit.Config{
		Deps:     deps,
		ExtraOps: func(reg *ops.Registry) {
			emailop.RegisterReply(reg, deps)
		},
		LoadTree: func(ctx context.Context, domainID int64, schemaID int) (*tree.Tree, error) {
			s, appErr := deps.GetSchemaById(domainID, schemaID)
			if appErr != nil {
				return nil, appErr
			}
			rawSchema := make([]map[string]any, len(s.Schema))
			for i, app := range s.Schema {
				rawSchema[i] = map[string]any(app)
			}
			return tree.Parse(s.Id, rawSchema)
		},
	})
	r.driver = kit.Driver
	r.sessionMgr = sessionmgr.New(kit.Coord, deps.RuntimeStateRepo(), deps.Log())

	return r
}

func (r *Router) GlobalVariable(domainId int64, name string) string {
	return r.fm.SchemaVariable(context.TODO(), domainId, name)
}

func (r *Router) Handle(rawConn model.Connection) error {
	go r.handle(rawConn)
	return nil
}

func (r *Router) handle(rawConn model.Connection) {
	// emailParser wraps EmailConnection to override ParseText for ${var} expansion
	// used by legacy global ops (httpRequest, set, etc.) via scope.Decode.
	conn := &emailParser{EmailConnection: rawConn.(model.EmailConnection)}

	s, err := r.fm.GetSchemaById(conn.DomainId(), conn.SchemaId())
	if err != nil {
		r.fm.Log().Error(fmt.Sprintf("email: conn %s schema error: %s", conn.Id(), err.Error()))
		return
	}

	autoLink, _ := r.fm.GetSystemSettings(conn.Context(), conn.DomainId(), model.SysAutoLinkMailToContact)
	if autoLink.BoolValue {
		r.linkContact(conn)
	}

	rawSchema := make([]map[string]any, len(s.Schema))
	for i, app := range s.Schema {
		rawSchema[i] = map[string]any(app)
	}
	tr, parseErr := tree.Parse(s.Id, rawSchema)
	if parseErr != nil {
		r.fm.Log().Error(fmt.Sprintf("email: conn %s parse error: %s", conn.Id(), parseErr.Error()))
		return
	}

	tags := make(map[string]string, len(tr.ByTag))
	for tag, node := range tr.ByTag {
		tags[tag] = node.ID
	}

	decorator := func(ctx context.Context) context.Context {
		return connctx.WithConnection(ctx, conn)
	}

	// Email is ephemeral: flows run to completion and never suspend.
	// RunSession still persists a runtime_state record for observability,
	// but sessionmgr.Watch is a no-op since emailParser.OnInboundMessage is a stub.
	if _, createErr := runtimekit.RunSession(nil, runtimekit.HandleConfig{
		ChannelName: "email",
		ChannelType: emailChannel,
		Conn:        conn,
		Tr:          tr,
		Tags:        tags,
		SchemaID:    s.Id,
		DomainID:    conn.DomainId(),
		AppID:       r.fm.AppID(),
		Repo:        r.fm.RuntimeStateRepo(),
		Driver:      r.driver,
		SessionMgr:  r.sessionMgr,
		Decorator:   decorator,
		Teardown:    func() {},
		Log:         r.fm.Log(),
	}); createErr != nil {
		r.fm.Log().Error(fmt.Sprintf("email: conn %s runtime error: %s", conn.Id(), createErr.Error()))
	}
}

func (r *Router) linkContact(conn model.EmailConnection) {
	email := conn.Email()
	if email == nil || len(email.From) == 0 {
		return
	}
	list, err := r.contacts.SearchNA(conn.Context(), &contacts.SearchContactsNARequest{
		DomainId: conn.DomainId(),
		Qin:      []string{"emails"},
		Q:        email.From[0],
		Size:     2,
		Fields:   []string{"id"},
	})
	if err != nil {
		conn.Log().Error("email linkContact: " + err.Error())
		return
	}
	if len(list.Data) == 1 {
		conn.Set(conn.Context(), model.Variables{"wbt_contact_id": list.Data[0].Id})
		cId := int64(0)
		fmt.Sscanf(list.Data[0].Id, "%d", &cId)
		if appErr := r.fm.MailSetContacts(conn.Context(), conn.DomainId(), conn.Id(), []int64{cId}); appErr != nil {
			conn.Log().Error("email mailSetContacts: " + appErr.Error())
		}
	}
}

// emailParser wraps model.EmailConnection and overrides ParseText so that
// legacy ops (via scope.Decode) can interpolate ${varName} from the email's
// connection variables.
type emailParser struct {
	model.EmailConnection
}

func (e *emailParser) ParseText(text string, _ ...model.ParseOption) string {
	return compileVar.ReplaceAllStringFunc(text, func(varName string) string {
		r := compileVar.FindStringSubmatch(varName)
		if len(r) > 0 {
			out, _ := e.Get(r[1])
			return out
		}
		return varName
	})
}

// OnInboundMessage satisfies sessionmgr.Connection. Email connections are
// ephemeral and never receive inbound messages after flow start.
func (e *emailParser) OnInboundMessage(_ func(string)) func() {
	return func() {}
}
