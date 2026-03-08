package handler

import (
	"testing"

	"git.neolidy.top/neo/storybook/internal/service"
)

func TestCanSprintTransit(t *testing.T) {
	cases := []struct {
		from string
		to   string
		ok   bool
	}{
		{"planned", "active", true},
		{"active", "completed", true},
		{"completed", "planned", false},
		{"planned", "completed", true},
	}

	for _, tc := range cases {
		if got := service.Workflow.CanSprintTransit(tc.from, tc.to); got != tc.ok {
			t.Fatalf("CanSprintTransit(%s,%s)=%v want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}

func TestCanBugTransit(t *testing.T) {
	cases := []struct {
		from string
		to   string
		ok   bool
	}{
		{"open", "in_progress", true},
		{"in_progress", "resolved", true},
		{"resolved", "closed", true},
		{"closed", "open", false},
	}

	for _, tc := range cases {
		if got := service.Workflow.CanBugTransit(tc.from, tc.to); got != tc.ok {
			t.Fatalf("CanBugTransit(%s,%s)=%v want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}
