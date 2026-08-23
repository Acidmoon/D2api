package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnonymizeLeaderboardName(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		userID   int64
		expected string
	}{
		{name: "email", email: "alice@example.com", userID: 12345, expected: "a***@e***.com"},
		{name: "short local", email: "a@b.co", userID: 1, expected: "a***@b***.co"},
		{name: "missing email", email: "", userID: 123456, expected: "User #3456"},
		{name: "invalid email", email: "no-at-sign", userID: 42, expected: "User #42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, anonymizeLeaderboardName(tt.email, tt.userID))
		})
	}
}
