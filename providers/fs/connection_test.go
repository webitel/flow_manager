package fs

import (
	"slices"
	"testing"

	"github.com/webitel/flow_manager/providers/fs/eventsocket"
)

func newEventsocketEvent(t *testing.T, args map[string]any) *eventsocket.Event {
	t.Helper()

	return &eventsocket.Event{
		Header: args,
	}
}

func TestConnection_initTransferExportVariables(t *testing.T) {
	t.Parallel()

	header := newEventsocketEvent(t, map[string]any{
		"variable_usr_hello":   "hello",
		"variable_usr_antonio": "banderas",
		"variable_cc_queue":    "inbound",
	})

	tests := []struct {
		name       string
		input      *eventsocket.Event
		connection *Connection
		want       []string
	}{{
		name:  "non_transfer",
		input: header,
		connection: &Connection{
			isBlindTransferQueue: false,
			transferAgent:        0,
		},
		want: []string{},
	}, {
		name:       "transfer_agent",
		input:      header,
		connection: &Connection{transferAgent: 1, isBlindTransferQueue: false},
		want:       []string{"hello", "antonio"},
	}, {
		name:       "transfer_queue",
		input:      header,
		connection: &Connection{isBlindTransferQueue: true, transferAgent: 0},
		want:       []string{"hello", "antonio"},
	}, {
		name:       "transfer_with_nil_header",
		input:      &eventsocket.Event{Header: nil},
		connection: &Connection{isBlindTransferQueue: true, transferAgent: 1},
		want:       nil,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.connection.initTransferExportVariables(tt.input)

			slices.Sort(tt.want)
			slices.Sort(tt.connection.exportVariables)

			if !slices.Equal(tt.want, tt.connection.exportVariables) {
				t.Errorf("Connection.initTransferExportVariables() got: %#v want%#v", tt.connection.exportVariables, tt.want)
			}
		})
	}
}
