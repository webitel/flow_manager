package call

import "testing"

func TestRecordSessionArgNormalize(t *testing.T) {
	tests := []struct {
		name      string
		in        RecordSessionArg
		wantType  string
		wantForce bool
	}{
		{name: "video format forces", in: RecordSessionArg{Type: "mp4"}, wantType: "mp4", wantForce: true},
		{name: "video format upper case", in: RecordSessionArg{Type: "MP4"}, wantType: "MP4", wantForce: true},
		{name: "video format with spaces", in: RecordSessionArg{Type: " mp4 "}, wantType: " mp4 ", wantForce: true},
		{name: "audio format keeps dedupe", in: RecordSessionArg{Type: "mp3"}, wantType: "mp3", wantForce: false},
		{name: "empty format defaults to audio", in: RecordSessionArg{}, wantType: defaultRecordSessionFormat, wantForce: false},
		{name: "explicit force kept", in: RecordSessionArg{Type: "mp3", Force: true}, wantType: "mp3", wantForce: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			arg := tc.in
			arg.Normalize()

			if arg.Type != tc.wantType {
				t.Fatalf("Type = %q, want %q", arg.Type, tc.wantType)
			}

			req := arg.ToRequest()
			if req.Force != tc.wantForce {
				t.Fatalf("ToRequest().Force = %v, want %v", req.Force, tc.wantForce)
			}
		})
	}
}
