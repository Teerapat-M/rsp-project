package redact

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEntrypointSedRedactsMigrateOutput(t *testing.T) {
	if _, err := exec.LookPath("sed"); err != nil {
		t.Skip("sed not on PATH")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	script := filepath.Join(filepath.Dir(file), "..", "..", "redact-log.sed")
	in := strings.Join([]string{
		`error: failed to open database, "postgres://rsp:` + leaked + `@db:5432/rsp?sslmode=disable": pq: password authentication failed for user "rsp"`,
		`error: failed to open database, "postgresql://leakuser:` + leaked + `@db:5432/rsp": FATAL: password authentication failed for user 'leakuser'`,
		`password=` + leaked,
	}, "\n")
	cmd := exec.Command("sed", "-f", script)
	cmd.Stdin = strings.NewReader(in)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, secret := range []string{leaked, "leakuser", `user "rsp"`, "user 'leakuser'"} {
		if strings.Contains(got, secret) {
			t.Fatalf("leaked %q in %s", secret, got)
		}
	}
	if !strings.Contains(got, "postgres://***@") || !strings.Contains(got, "postgresql://***@") {
		t.Fatalf("expected masked URLs, got %s", got)
	}
	if !strings.Contains(got, `for user "***"`) || !strings.Contains(got, "for user '"+mask+"'") {
		t.Fatalf("expected masked for user, got %s", got)
	}
}
