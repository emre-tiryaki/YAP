package database

import (
	"path/filepath"
	"testing"
)

func TestNewWithDSN_Succes_InMemory(t *testing.T) {
	dsn := "file::memory:?cache?shared"

	db, err := NewWithDSN(dsn)
	if err != nil {
		t.Fatalf("Database Connection Not Opened: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Errorf("Database Ping Failed: %v", err)
	}
}

func TestNewWithDSN_Succes_Tempfile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_app.db")

	db, err := NewWithDSN(dbPath)
	if err != nil {
		t.Fatalf("Database Connection Not Opened: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Errorf("Database Ping Failed: %v", err)
	}
}

func TestNewWithDSN_Failuer_Invalid_DSN(t *testing.T) {
	invalidDSN := "/non_existent_directory/invalid_path/app.db"

	db, err := NewWithDSN(invalidDSN)
	if err == nil {
		db.Close()
		t.Fatalf("It should have given an error but it returned nil")
	}
}
