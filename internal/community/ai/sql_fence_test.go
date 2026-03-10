// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package ai

import "testing"

func TestStripSQLCodeFence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "SELECT 1", "SELECT 1"},
		{"sql block", "```sql\nSELECT 1\n```", "SELECT 1"},
		{"generic block", "```\nSELECT 1\n```", "SELECT 1"},
		{"with newlines", "  \n```sql\nSELECT * FROM t\n```  \n", "SELECT * FROM t"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripSQLCodeFence(tt.in)
			if got != tt.want {
				t.Errorf("StripSQLCodeFence() = %q, want %q", got, tt.want)
			}
		})
	}
}
