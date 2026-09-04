package models

import "testing"

func TestDefaultEnabledByChannel(t *testing.T) {
	tests := []struct {
		name    string
		nType   NotificationType
		channel NotificationChannel
		want    bool
	}{
		{"assignment in app", NotificationTypeAssignment, NotificationChannelInApp, true},
		{"assignment email", NotificationTypeAssignment, NotificationChannelEmail, true},
		{"assignment push", NotificationTypeAssignment, NotificationChannelPush, false},
		{"new reply in app", NotificationTypeNewReply, NotificationChannelInApp, false},
		{"new reply email", NotificationTypeNewReply, NotificationChannelEmail, false},
		{"new reply push", NotificationTypeNewReply, NotificationChannelPush, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultEnabled(tt.nType, tt.channel); got != tt.want {
				t.Fatalf("DefaultEnabled(%q, %q) = %v, want %v", tt.nType, tt.channel, got, tt.want)
			}
		})
	}
}
