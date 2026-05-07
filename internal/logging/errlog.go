package logging

import "log/slog"

// LogIfErr logs err at Error level when non-nil and returns immediately when nil.
// Use to surface errors that would otherwise be silently dropped (e.g., audit log
// writes, secondary count queries, best-effort updates) without changing the
// calling function's signature or control flow.
//
// attrs follow slog convention: alternating key/value pairs, or pre-built
// slog.Attr values.
func LogIfErr(err error, msg string, attrs ...any) {
	if err == nil {
		return
	}
	args := make([]any, 0, len(attrs)+2)
	args = append(args, attrs...)
	args = append(args, "error", err)
	slog.Error(msg, args...)
}
