package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateDatabaseURLFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "database-url")
	if err := os.WriteFile(path, []byte("postgres://fake:fake@localhost/test"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_URL_FILE", path)
	t.Setenv("SENTINEL_ENV", "development")
	t.Setenv("SENTINEL_AUTH_MODE", "development")
	cfg, err := Load()
	if err != nil || cfg.DatabaseURL != "postgres://fake:fake@localhost/test" {
		t.Fatal("file load", err)
	}
	t.Setenv("DATABASE_URL", "postgres://other")
	if _, err := Load(); err == nil {
		t.Fatal("ambiguous sources accepted")
	}
	t.Setenv("DATABASE_URL", "")
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("public file accepted")
	}
}
