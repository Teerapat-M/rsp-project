package migrations_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// Matches golang-migrate's filename parser: {version}_{title}.{direction}.{ext}
var migrateName = regexp.MustCompile(`^([0-9]+)_(.*)\.(up|down)\.sql$`)

func TestUniqueVersion(t *testing.T) {
	t.Parallel()
	if err := uniqueMigrationVersions(migrationsDir(t)); err != nil {
		t.Fatal(err)
	}
}

func TestDuplicateMigrationVersionsFail(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, name := range []string{
		"000001_init.up.sql",
		"000001_init.down.sql",
		"000001_stale.up.sql",
		"000001_stale.down.sql",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("--\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	err := uniqueMigrationVersions(dir)
	if err == nil {
		t.Fatal("expected duplicate version error")
	}
	if !strings.Contains(err.Error(), "000001") {
		t.Fatalf("want version 000001 in error, got %v", err)
	}
}

func uniqueMigrationVersions(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type key struct {
		version   string
		direction string
	}
	seen := map[key]string{}
	var dups []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		m := migrateName.FindStringSubmatch(e.Name())
		if m == nil {
			return fmt.Errorf("migration filename must be {version}_{name}.{up|down}.sql: %s", e.Name())
		}
		k := key{version: m[1], direction: m[3]}
		if prev, ok := seen[k]; ok {
			dups = append(dups, fmt.Sprintf("version %s %s: %s and %s", m[1], m[3], prev, e.Name()))
			continue
		}
		seen[k] = e.Name()
	}
	if len(dups) > 0 {
		sort.Strings(dups)
		return fmt.Errorf("duplicate migration file(s):\n  %s", strings.Join(dups, "\n  "))
	}
	return nil
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	return filepath.Dir(file)
}
