package repositories

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"pr-1/internal/db"
	"pr-1/internal/models"
)

// SettingsRepository — доступ к настройкам приложения.
type SettingsRepository struct {
	db *db.DB
}

func NewSettingsRepository(database *db.DB) *SettingsRepository {
	return &SettingsRepository{db: database}
}

func (r *SettingsRepository) GetAppSettings(ctx context.Context) (*models.AppSettings, error) {
	var raw []byte
	err := r.db.Pool.QueryRow(ctx, `SELECT value FROM settings WHERE key='app'`).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return &models.AppSettings{AutoNumberOrders: true}, nil
	}
	if err != nil {
		return nil, err
	}
	var s models.AppSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SettingsRepository) SaveAppSettings(ctx context.Context, s models.AppSettings) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = r.db.Pool.Exec(ctx,
		`INSERT INTO settings (key, value) VALUES ('app', $1)
		 ON CONFLICT (key) DO UPDATE SET value=$1`, b)
	return err
}
