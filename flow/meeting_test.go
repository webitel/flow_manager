package flow

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

// --- meeting mock ---

type mockMeetingClient struct {
	createFn func(ctx context.Context, domainId int64, title string, expireSec int, basePath string, vars map[string]string) (string, error)
}

func (m *mockMeetingClient) Create(ctx context.Context, domainId int64, title string, expireSec int, basePath string, vars map[string]string) (string, error) {
	return m.createFn(ctx, domainId, title, expireSec, basePath, vars)
}

func (m *mockMeetingClient) Get(_ context.Context, _ string) (map[string]string, error) {
	return nil, nil
}

var _ domainmeeting.Client = (*mockMeetingClient)(nil)

// --- connection mock ---

type mockConn struct {
	domainId int64
	setFn    func(ctx context.Context, vars model.Variables) (model.Response, *model.AppError)
	logger   *wlog.Logger
}

func newMockConn(domainId int64, setFn func(ctx context.Context, vars model.Variables) (model.Response, *model.AppError)) *mockConn {
	return &mockConn{
		domainId: domainId,
		setFn:    setFn,
		logger:   wlog.NewLogger(&wlog.LoggerConfiguration{}),
	}
}

func (c *mockConn) Type() model.ConnectionType                           { return model.ConnectionTypeCall }
func (c *mockConn) Id() string                                           { return "test-id" }
func (c *mockConn) NodeId() string                                       { return "node-id" }
func (c *mockConn) DomainId() int64                                      { return c.domainId }
func (c *mockConn) Context() context.Context                             { return context.Background() }
func (c *mockConn) Get(_ string) (string, bool)                          { return "", false }
func (c *mockConn) Variables() map[string]string                         { return nil }
func (c *mockConn) Log() *wlog.Logger                                    { return c.logger }
func (c *mockConn) Close() *model.AppError                               { return nil }
func (c *mockConn) ParseText(text string, _ ...model.ParseOption) string { return text }
func (c *mockConn) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	return c.setFn(ctx, vars)
}

var _ model.Connection = (*mockConn)(nil)

// --- router mock (needed by flow.New) ---

type mockFlowRouter struct{}

func (r *mockFlowRouter) Handle(_ model.Connection) *model.AppError { return nil }
func (r *mockFlowRouter) GlobalVariable(_ int64, _ string) string   { return "" }

// --- helpers ---

func newTestScope(conn model.Connection) *Flow {
	return New(&mockFlowRouter{}, Config{Conn: conn})
}

// --- tests ---

func TestCreateMeeting_MissingSetVar(t *testing.T) {
	conn := newMockConn(1, nil)
	r := &router{meeting: &mockMeetingClient{}}

	_, err := r.createMeeting(context.Background(), newTestScope(conn), conn, map[string]interface{}{
		"title": "standup",
	})

	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "setVar")
}

func TestCreateMeeting_ClientError(t *testing.T) {
	conn := newMockConn(1, nil)
	r := &router{
		meeting: &mockMeetingClient{
			createFn: func(_ context.Context, _ int64, _ string, _ int, _ string, _ map[string]string) (string, error) {
				return "", errors.New("grpc unavailable")
			},
		},
	}

	_, appErr := r.createMeeting(context.Background(), newTestScope(conn), conn, map[string]interface{}{
		"setVar": "meetingUrl",
	})

	require.NotNil(t, appErr)
	assert.Contains(t, appErr.Error(), "grpc unavailable")
}

func TestCreateMeeting_Success(t *testing.T) {
	const wantURL = "https://meet.example.com/abc123"
	var gotVars model.Variables

	conn := newMockConn(42, func(_ context.Context, vars model.Variables) (model.Response, *model.AppError) {
		gotVars = vars
		return model.CallResponseOK, nil
	})
	r := &router{
		meeting: &mockMeetingClient{
			createFn: func(_ context.Context, domainId int64, title string, expireSec int, _ string, _ map[string]string) (string, error) {
				assert.Equal(t, int64(42), domainId)
				assert.Equal(t, "standup", title)
				assert.Equal(t, 3600, expireSec)
				return wantURL, nil
			},
		},
	}

	res, appErr := r.createMeeting(context.Background(), newTestScope(conn), conn, map[string]interface{}{
		"setVar":    "meetingUrl",
		"title":     "standup",
		"expireSec": 3600,
	})

	require.Nil(t, appErr)
	assert.Equal(t, model.CallResponseOK, res)
	assert.Equal(t, wantURL, gotVars["meetingUrl"])
}
