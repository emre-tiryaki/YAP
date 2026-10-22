package database

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func New() (*sql.DB, error) {
	return NewWithDSN("./app.db");
}

func NewWithDSN(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil;
}