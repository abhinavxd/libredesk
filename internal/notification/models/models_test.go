package models

import "testing"

func TestDefaultEnabled(t *testing.T) {
	tests := []struct {
		name  string
		nType NotificationType
		want  bool
	}{
		{"assignment", NotificationTypeAssignment, true},
		{"mention", NotificationTypeMention, true},
		{"new reply", NotificationTypeNewReply, false},
		{"new reply participating", NotificationTypeNewReplyParticipating, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultEnabled(tt.nType); got != tt.want {
				t.Fatalf("DefaultEnabled(%q) = %v, want %v", tt.nType, got, tt.want)
			}
		})
	}
}
