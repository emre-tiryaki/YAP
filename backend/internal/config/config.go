package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath string
	Port   string
}

func Load() (Config, error) {
	baseDir, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}
	if path, ok := findUp(".env"); ok {
		if err := godotenv.Load(path); err != nil {
			return Config{}, fmt.Errorf("load %s: %w", path, err)
		}
		baseDir = filepath.Dir(path)
	}
	return Config{
		DBPath: resolve(baseDir, getenv("DB_PATH", "./app.db")),
		Port:   getenv("PORT", "8080"),
	}, nil
}

func resolve(baseDir, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(baseDir, p)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func findUp(name string) (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return "", false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}