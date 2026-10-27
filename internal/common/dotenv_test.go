package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDotEnvIfPresent(t *testing.T) {
	t.Setenv("LANDAPTER_DOTENV_TEST_EXISTING", "from-process")
	unsetEnvForTest(t, "LANDAPTER_DOTENV_TEST_QUOTED")
	unsetEnvForTest(t, "POSTGRES_USER")
	unsetEnvForTest(t, "POSTGRES_PASSWORD")
	unsetEnvForTest(t, "DB_USER")
	unsetEnvForTest(t, "DB_PASSWORD")

	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\n" +
		"LANDAPTER_DOTENV_TEST_EXISTING=from-file\n" +
		"LANDAPTER_DOTENV_TEST_QUOTED=\"hello world\"\n" +
		"POSTGRES_USER=landapter\n" +
		"POSTGRES_PASSWORD=abc123\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	if err := LoadDotEnvIfPresent(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("LANDAPTER_DOTENV_TEST_EXISTING"); got != "from-process" {
		t.Fatalf("process environment should win, got %q", got)
	}
	if got := os.Getenv("LANDAPTER_DOTENV_TEST_QUOTED"); got != "hello world" {
		t.Fatalf("quoted value = %q, want %q", got, "hello world")
	}
	if got := os.Getenv("DB_USER"); got != "landapter" {
		t.Fatalf("DB_USER alias = %q, want landapter", got)
	}
	if got := os.Getenv("DB_PASSWORD"); got != "abc123" {
		t.Fatalf("DB_PASSWORD alias = %q, want generated password", got)
	}
}

func TestLoadDotEnvIfPresentMissingFile(t *testing.T) {
	if err := LoadDotEnvIfPresent(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Fatalf("missing .env should be allowed, got %v", err)
	}
}

func TestLoadDotEnvRejectsInvalidKeyWithoutLeakingValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("BAD KEY=do-not-print-this-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	err := LoadDotEnvIfPresent(path)
	if err == nil {
		t.Fatal("expected invalid key to be rejected")
	}
	if got := err.Error(); strings.Contains(got, "do-not-print-this-secret") {
		t.Fatal("error message leaked an environment value")
	}
}

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()
	old, exists := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func TestLoadDotEnvMapsDatabaseVariablesWithoutFile(t *testing.T) {
	t.Setenv("POSTGRES_USER", "process-user")
	t.Setenv("POSTGRES_PASSWORD", "process-password")
	unsetEnvForTest(t, "DB_USER")
	unsetEnvForTest(t, "DB_PASSWORD")

	if err := LoadDotEnvIfPresent(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("DB_USER"); got != "process-user" {
		t.Fatalf("DB_USER alias = %q, want process-user", got)
	}
	if got := os.Getenv("DB_PASSWORD"); got != "process-password" {
		t.Fatalf("DB_PASSWORD alias = %q, want process-password", got)
	}
}
