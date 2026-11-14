package repository

import (
	"context"
	"fmt"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
)

type SettingsRepository struct{}

func NewSettingsRepository() *SettingsRepository {
	return &SettingsRepository{}
}

// All returns the stored settings of a user as raw JSON values by key.
func (*SettingsRepository) All(ctx context.Context, q database.DBTX, userID string) (map[string]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT key, value FROM user_settings WHERE user_id = ?`, userID)
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	return values, nil
}

func (*SettingsRepository) Put(ctx context.Context, q database.DBTX, userID, key, value, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT (user_id, key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		userID, key, value, now,
	)
	if err != nil {
		return fmt.Errorf("put setting: %w", err)
	}
	return nil
}
