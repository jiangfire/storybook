package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunHelpFlag(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := run([]string{arg}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit code 0 for %q, got %d", arg, code)
		}
		if !strings.Contains(stdout.String(), "子命令:") {
			t.Fatalf("expected usage output for %q, got %q", arg, stdout.String())
		}
	}
}

func TestRunVersionFlag(t *testing.T) {
	for _, arg := range []string{"version", "-v", "--version"} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := run([]string{arg}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit code 0 for %q, got %d", arg, code)
		}
		if !strings.Contains(stdout.String(), version) {
			t.Fatalf("expected version output for %q, got %q", arg, stdout.String())
		}
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"foobar"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "未知子命令: foobar") {
		t.Fatalf("expected unknown command error, got %q", stderr.String())
	}
}

func TestRunServerHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{cmdServer, "-h"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stderr.String(), "启动 storybook HTTP 服务器") {
		t.Fatalf("expected server help output, got %q", stderr.String())
	}
}

func TestRunBootstrapAdminRequiresEmail(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{cmdBootstrapAdmin}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "缺少必填参数: --email") {
		t.Fatalf("expected missing email error, got %q", stderr.String())
	}
}

func TestRunBootstrapAdminDoesNotRequireJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("DB_DSN", "file:bootstrap_admin_cli_test?mode=memory&cache=shared")
	t.Setenv("DB_AUTO_MIGRATE", "true")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{cmdBootstrapAdmin, "--email", "admin@example.com", "--password", "Admin1234"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "admin created:") {
		t.Fatalf("expected success output, got %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "JWT_SECRET must be set") {
		t.Fatalf("bootstrap should not require JWT_SECRET, stderr=%q", stderr.String())
	}
}
