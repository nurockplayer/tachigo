package models

import "testing"

func TestClaimStatusValues(t *testing.T) {
	tests := []struct {
		name string
		got  ClaimStatus
		want string
	}{
		{name: "pending", got: ClaimStatusPending, want: "pending"},
		{name: "broadcast", got: ClaimStatusBroadcast, want: "broadcast"},
		{name: "confirmed", got: ClaimStatusConfirmed, want: "confirmed"},
		{name: "failed", got: ClaimStatusFailed, want: "failed"},
	}

	for _, tc := range tests {
		if string(tc.got) != tc.want {
			t.Fatalf("%s: want %q, got %q", tc.name, tc.want, tc.got)
		}
	}
}
