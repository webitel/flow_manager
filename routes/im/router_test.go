package im

import (
	"slices"
	"testing"

	"github.com/webitel/flow_manager/model"
)

// recordingDialog records the natural-end calls; any other method panics on the nil embed.
type recordingDialog struct {
	model.IMDialog

	terminating bool
	completeID  string
	calls       []string
}

func (d *recordingDialog) IsTerminating() bool { return d.terminating }
func (d *recordingDialog) MarkCompleted()      { d.calls = append(d.calls, "mark-completed") }
func (d *recordingDialog) CompleteId() string  { return d.completeID }
func (d *recordingDialog) Complete(id string)  { d.calls = append(d.calls, "complete:"+id) }
func (d *recordingDialog) ResumeParent()       { d.calls = append(d.calls, "resume-parent") }

func TestFinishNaturally(t *testing.T) {
	tests := []struct {
		name      string
		dialog    *recordingDialog
		wantOK    bool
		wantCalls []string
	}{
		{
			name:      "nested schema",
			dialog:    &recordingDialog{completeID: "member-1"},
			wantOK:    true,
			wantCalls: []string{"mark-completed", "complete:member-1", "resume-parent"},
		},
		{
			name:      "no bot control entry",
			dialog:    &recordingDialog{},
			wantOK:    true,
			wantCalls: []string{"mark-completed", "resume-parent"},
		},
		{
			name:   "terminated meanwhile",
			dialog: &recordingDialog{terminating: true, completeID: "member-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := finishNaturally(tt.dialog); got != tt.wantOK {
				t.Errorf("finishNaturally = %v, want %v", got, tt.wantOK)
			}

			if !slices.Equal(tt.dialog.calls, tt.wantCalls) {
				t.Errorf("calls = %v, want %v", tt.dialog.calls, tt.wantCalls)
			}
		})
	}
}
