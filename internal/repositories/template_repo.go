package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"pr-1/internal/db"
	"pr-1/internal/models"
)

// TemplateRepository — доступ к шаблонам.
type TemplateRepository struct {
	db *db.DB
}

func NewTemplateRepository(database *db.DB) *TemplateRepository {
	return &TemplateRepository{db: database}
}

func (r *TemplateRepository) List(ctx context.Context, kind string) ([]models.Template, error) {
	query := `SELECT id, kind, name, coalesce(description,''), content, style, created_at FROM templates`
	args := []interface{}{}
	if kind != "" {
		query += ` WHERE kind=$1`
		args = append(args, kind)
	}
	query += ` ORDER BY created_at`
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.Template{}
	for rows.Next() {
		var t models.Template
		var content, style []byte
		if err := rows.Scan(&t.ID, &t.Kind, &t.Name, &t.Description, &content, &style, &t.CreatedAt); err != nil {
			return nil, err
		}
		if len(content) > 0 {
			_ = json.Unmarshal(content, &t.Content)
		}
		if len(style) > 0 {
			_ = json.Unmarshal(style, &t.Style)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *TemplateRepository) Get(ctx context.Context, id string) (*models.Template, error) {
	var t models.Template
	var content, style []byte
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, kind, name, coalesce(description,''), content, style, created_at FROM templates WHERE id=$1`, id).
		Scan(&t.ID, &t.Kind, &t.Name, &t.Description, &content, &style, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("шаблон не найден")
	}
	if err != nil {
		return nil, err
	}
	if len(content) > 0 {
		_ = json.Unmarshal(content, &t.Content)
	}
	if len(style) > 0 {
		_ = json.Unmarshal(style, &t.Style)
	}
	return &t, nil
}

func (r *TemplateRepository) Create(ctx context.Context, t models.Template) (*models.Template, error) {
	content, err := json.Marshal(t.Content)
	if err != nil {
		return nil, err
	}
	style, err := json.Marshal(t.Style)
	if err != nil {
		return nil, err
	}
	err = r.db.Pool.QueryRow(ctx,
		`INSERT INTO templates (kind, name, description, content, style) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`, t.Kind, t.Name, t.Description, content, style).
		Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TemplateRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Pool.Exec(ctx, `DELETE FROM templates WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("шаблон не найден")
	}
	return nil
}
