package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath string
	// ResourcesPath is where resource files are kept on disk.
	ResourcesPath string
	Port          string
	DisplayName   string
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
	dbPath := resolve(baseDir, getenv("DB_PATH", "./app.db"))
	resourcesPath := filepath.Join(filepath.Dir(dbPath), "resources")
	if v := os.Getenv("RESOURCES_PATH"); v != "" {
		resourcesPath = resolve(baseDir, v)
	}
	return Config{
		DBPath:        dbPath,
		ResourcesPath: resourcesPath,
		Port:          getenv("BACKEND_PORT", "8080"),
		DisplayName:   getenv("DISPLAY_NAME", defaultDisplayName()),
	}, nil
}

func defaultDisplayName() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return "user"
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
