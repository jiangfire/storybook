package handler

import (
	"testing"

	"git.neolidy.top/neo/storybook/internal/service"
)

func TestCanTransit(t *testing.T) {
	cases := []struct {
		from string
		to   string
		ok   bool
	}{
		{"backlog", "ready", true},
		{"ready", "done", false},
		{"in_progress", "test", true},
		{"done", "backlog", false},
		{"test", "done", true},
	}

	for _, tc := range cases {
		if got := service.Workflow.CanStoryTransit(tc.from, tc.to); got != tc.ok {
			t.Fatalf("CanStoryTransit(%s,%s)=%v want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}

func TestNormalizeAC(t *testing.T) {
	items := []createStoryACItem{
		{ID: "", Description: "b", Order: 2},
		{ID: "", Description: "a", Order: 1},
	}

	out := normalizeAC(items)
	if len(out) != 2 {
		t.Fatalf("expected 2 items, got %d", len(out))
	}
	if out[0].Order != 1 || out[1].Order != 2 {
		t.Fatalf("expected sorted order 1,2 got %d,%d", out[0].Order, out[1].Order)
	}
	if out[0].ID == "" || out[1].ID == "" {
		t.Fatalf("expected generated AC ids")
	}
}
