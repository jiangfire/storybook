package logging

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestLogIfErr_NilDoesNothing(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	LogIfErr(nil, "should not log", "key", "value")

	if buf.Len() != 0 {
		t.Fatalf("expected no log output, got %q", buf.String())
	}
}

func TestLogIfErr_LogsWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	want := errors.New("boom")
	LogIfErr(want, "thing failed", "story_id", uint(42))

	out := buf.String()
	if !strings.Contains(out, "thing failed") {
		t.Errorf("expected message in output, got %q", out)
	}
	if !strings.Contains(out, "story_id=42") {
		t.Errorf("expected story_id attr, got %q", out)
	}
	if !strings.Contains(out, `error=boom`) {
		t.Errorf("expected error attr, got %q", out)
	}
	if !strings.Contains(out, "level=ERROR") {
		t.Errorf("expected ERROR level, got %q", out)
	}
}
