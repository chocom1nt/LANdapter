package common

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LoadProjectDotEnv looks for .env in the working directory, next to the
// executable, and one directory above the executable (the usual bin/ layout).
// It lets users start Master and lndp without manually exporting secrets.
func LoadProjectDotEnv() error {
	candidates := []string{".env"}
	if executable, err := os.Executable(); err == nil {
		dir := filepath.Dir(executable)
		candidates = append(candidates, filepath.Join(dir, ".env"), filepath.Join(filepath.Dir(dir), ".env"))
	}
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		if _, err := os.Stat(candidate); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("inspect environment file %q: %w", candidate, err)
		}
		return LoadDotEnvIfPresent(candidate)
	}
	// This is also where aliases from process-level POSTGRES_* variables are
	// mapped when no local .env file exists.
	return LoadDotEnvIfPresent(".env")
}

// LoadDotEnvIfPresent loads KEY=VALUE pairs from path when the file exists.
// Variables already present in the process environment always take precedence.
// Secret values are never included in returned error messages.
func LoadDotEnvIfPresent(path string) error {
	file, err := os.Open(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("open environment file %q: %w", path, err)
	}
	if file != nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 4096), 1<<20)
		lineNumber := 0

		for scanner.Scan() {
			lineNumber++
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			key, rawValue, ok := strings.Cut(line, "=")
			key = strings.TrimSpace(key)
			if !ok || !validEnvKey(key) {
				return fmt.Errorf("invalid entry in %q at line %d; expected KEY=VALUE", path, lineNumber)
			}
			if _, exists := os.LookupEnv(key); exists {
				continue
			}

			value, err := parseDotEnvValue(strings.TrimSpace(rawValue))
			if err != nil {
				return fmt.Errorf("invalid value syntax in %q at line %d", path, lineNumber)
			}
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set environment variable %q: %w", key, err)
			}
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read environment file %q: %w", path, err)
		}
	}

	// Docker Compose uses POSTGRES_* while a locally run Master uses DB_*.
	// Respect explicit DB_* values, but map the generated local defaults.
	if value, ok := os.LookupEnv("POSTGRES_USER"); ok {
		if _, exists := os.LookupEnv("DB_USER"); !exists {
			if err := os.Setenv("DB_USER", value); err != nil {
				return fmt.Errorf("set DB_USER from POSTGRES_USER: %w", err)
			}
		}
	}
	if value, ok := os.LookupEnv("POSTGRES_PASSWORD"); ok {
		if _, exists := os.LookupEnv("DB_PASSWORD"); !exists {
			if err := os.Setenv("DB_PASSWORD", value); err != nil {
				return fmt.Errorf("set DB_PASSWORD from POSTGRES_PASSWORD: %w", err)
			}
		}
	}
	return nil
}

func validEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func parseDotEnvValue(raw string) (string, error) {
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		return strconv.Unquote(raw)
	}
	if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
		return raw[1 : len(raw)-1], nil
	}
	// For unquoted values, a comment starts only at a whitespace-prefixed '#'.
	if i := strings.Index(raw, " #"); i >= 0 {
		raw = strings.TrimSpace(raw[:i])
	}
	return raw, nil
}
