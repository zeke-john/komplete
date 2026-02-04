package config

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Config map[string]string

func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err == nil && dir != "" {
		return filepath.Join(dir, "komplete", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "komplete", "config.toml"), nil
}

func Load(path string) (Config, error) {
	cfg := Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		if key != "" {
			cfg[key] = value
		}
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(cfg))
	for k, v := range cfg {
		if v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(" = ")
		b.WriteString(quote(cfg[k]))
		b.WriteString("\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func AllowedKeys() []string {
	return []string{"model", "shell", "timeout", "cwd"}
}

func IsAllowedKey(key string) bool {
	for _, k := range AllowedKeys() {
		if k == key {
			return true
		}
	}
	return false
}

func quote(value string) string {
	escaped := strings.ReplaceAll(value, `"`, `\"`)
	return `"` + escaped + `"`
}
