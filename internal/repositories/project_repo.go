package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"pr-1/internal/db"
	"pr-1/internal/models"
)

// ProjectRepository — доступ к данным проектов.
type ProjectRepository struct {
	db *db.DB
}

func NewProjectRepository(database *db.DB) *ProjectRepository {
	return &ProjectRepository{db: database}
}

func (r *ProjectRepository) Create(ctx context.Context, dto models.ProjectDTO) (*models.Project, error) {
	content, err := json.Marshal(dto.Content)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации содержимого: %w", err)
	}
	status := dto.Status
	if status == "" {
		status = models.ProjectStatusDraft
	}
	var p models.Project
	err = r.db.Pool.QueryRow(ctx,
		`INSERT INTO projects (title, content, status) VALUES ($1, $2, $3)
		 RETURNING id, title, content, status, created_at, updated_at`,
		dto.Title, content, status).Scan(&p.ID, &p.Title, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepository) Update(ctx context.Context, id string, dto models.ProjectDTO) (*models.Project, error) {
	content, err := json.Marshal(dto.Content)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации содержимого: %w", err)
	}
	status := dto.Status
	if status == "" {
		status = models.ProjectStatusDraft
	}
	var p models.Project
	err = r.db.Pool.QueryRow(ctx,
		`UPDATE projects SET title=$2, content=$3, status=$4, updated_at=now()
		 WHERE id=$1 RETURNING id, title, content, status, created_at, updated_at`,
		id, dto.Title, content, status).Scan(&p.ID, &p.Title, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("проект не найден")
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepository) Get(ctx context.Context, id string) (*models.Project, error) {
	var p models.Project
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, title, content, status, created_at, updated_at FROM projects WHERE id=$1`, id).
		Scan(&p.ID, &p.Title, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("проект не найден")
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepository) List(ctx context.Context, filter string) ([]models.ProjectSummary, error) {
	query := `SELECT id, title, status, updated_at FROM projects`
	args := []interface{}{}
	if strings.TrimSpace(filter) != "" {
		query += ` WHERE title ILIKE $1`
		args = append(args, "%"+strings.TrimSpace(filter)+"%")
	}
	query += ` ORDER BY updated_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []models.ProjectSummary{}
	for rows.Next() {
		var s models.ProjectSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.Status, &s.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Pool.Exec(ctx, `DELETE FROM projects WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("проект не найден")
	}
	return nil
}
