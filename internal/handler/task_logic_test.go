package handler

import (
	"testing"

	"git.neolidy.top/neo/storybook/internal/service"
)

func TestCanTaskTransit(t *testing.T) {
	cases := []struct {
		from string
		to   string
		ok   bool
	}{
		{"todo", "in_progress", true},
		{"in_progress", "done", true},
		{"blocked", "done", true},
		{"done", "todo", false},
	}

	for _, tc := range cases {
		if got := service.Workflow.CanTaskTransit(tc.from, tc.to); got != tc.ok {
			t.Fatalf("CanTaskTransit(%s,%s)=%v want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}
