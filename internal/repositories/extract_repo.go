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

// ExtractRepository — доступ к данным выписок.
type ExtractRepository struct {
	db *db.DB
}

func NewExtractRepository(database *db.DB) *ExtractRepository {
	return &ExtractRepository{db: database}
}

func (r *ExtractRepository) Create(ctx context.Context, orderID, title string, content models.DocumentContent) (*models.Extract, error) {
	b, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	var e models.Extract
	err = r.db.Pool.QueryRow(ctx,
		`INSERT INTO extracts (order_id, title, content) VALUES ($1, $2, $3)
		 RETURNING id, order_id, title, content, created_at`,
		orderID, title, b).Scan(&e.ID, &e.OrderID, &e.Title, &e.Content, &e.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "extracts_order_id_fkey") {
			return nil, fmt.Errorf("приказ не найден")
		}
		return nil, err
	}
	return &e, nil
}

func (r *ExtractRepository) Get(ctx context.Context, id string) (*models.Extract, error) {
	var e models.Extract
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, order_id, title, content, created_at FROM extracts WHERE id=$1`, id).
		Scan(&e.ID, &e.OrderID, &e.Title, &e.Content, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("выписка не найдена")
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *ExtractRepository) List(ctx context.Context, filter string) ([]models.ExtractSummary, error) {
	query := `SELECT e.id, e.order_id, e.title,
	                 o.order_number || ' от ' || to_char(o.order_date, 'DD.MM.YYYY'),
	                 e.created_at
	          FROM extracts e JOIN orders o ON o.id = e.order_id`
	args := []interface{}{}
	if strings.TrimSpace(filter) != "" {
		query += ` WHERE e.title ILIKE $1`
		args = append(args, "%"+strings.TrimSpace(filter)+"%")
	}
	query += ` ORDER BY e.created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []models.ExtractSummary{}
	for rows.Next() {
		var s models.ExtractSummary
		if err := rows.Scan(&s.ID, &s.OrderID, &s.Title, &s.OrderInfo, &s.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *ExtractRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Pool.Exec(ctx, `DELETE FROM extracts WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("выписка не найдена")
	}
	return nil
}
