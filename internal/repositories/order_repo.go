package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"pr-1/internal/db"
	"pr-1/internal/models"
)

// OrderRepository — доступ к данным приказов.
type OrderRepository struct {
	db *db.DB
}

func NewOrderRepository(database *db.DB) *OrderRepository {
	return &OrderRepository{db: database}
}

func (r *OrderRepository) Create(ctx context.Context, projectID, number, date string, content models.DocumentContent) (*models.Order, error) {
	b, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	var o models.Order
	var orderDate time.Time
	var projectIDOut *string
	err = r.db.Pool.QueryRow(ctx,
		`INSERT INTO orders (project_id, order_number, order_date, content, status)
		 VALUES ($1, $2, $3, $4, 'issued')
		 RETURNING id, project_id, order_number, order_date, content, status, created_at, updated_at`,
		nullUUID(projectID), number, date, b).
		Scan(&o.ID, &projectIDOut, &o.OrderNumber, &orderDate, &o.Content, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "orders_number_unique") {
			return nil, fmt.Errorf("приказ с номером %s уже существует", number)
		}
		return nil, err
	}
	if projectIDOut != nil {
		o.ProjectID = *projectIDOut
	}
	o.OrderDate = orderDate.Format("2006-01-02")
	return &o, nil
}

func (r *OrderRepository) Update(ctx context.Context, id string, dto models.OrderDTO) (*models.Order, error) {
	b, err := json.Marshal(dto.Content)
	if err != nil {
		return nil, err
	}
	var o models.Order
	var orderDate time.Time
	var projectIDOut *string
	err = r.db.Pool.QueryRow(ctx,
		`UPDATE orders SET order_number=$2, order_date=$3, content=$4, updated_at=now()
		 WHERE id=$1
		 RETURNING id, project_id, order_number, order_date, content, status, created_at, updated_at`,
		id, dto.OrderNumber, dto.OrderDate, b).
		Scan(&o.ID, &projectIDOut, &o.OrderNumber, &orderDate, &o.Content, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("приказ не найден")
	}
	if err != nil {
		if strings.Contains(err.Error(), "orders_number_unique") {
			return nil, fmt.Errorf("приказ с номером %s уже существует", dto.OrderNumber)
		}
		return nil, err
	}
	if projectIDOut != nil {
		o.ProjectID = *projectIDOut
	}
	o.OrderDate = orderDate.Format("2006-01-02")
	return &o, nil
}

func (r *OrderRepository) Get(ctx context.Context, id string) (*models.Order, error) {
	var o models.Order
	var orderDate time.Time
	var projectIDOut *string
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, project_id, order_number, order_date, content, status, created_at, updated_at
		 FROM orders WHERE id=$1`, id).
		Scan(&o.ID, &projectIDOut, &o.OrderNumber, &orderDate, &o.Content, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("приказ не найден")
	}
	if err != nil {
		return nil, err
	}
	if projectIDOut != nil {
		o.ProjectID = *projectIDOut
	}
	o.OrderDate = orderDate.Format("2006-01-02")
	return &o, nil
}

func (r *OrderRepository) List(ctx context.Context, filter string) ([]models.OrderSummary, error) {
	query := `SELECT o.id, o.order_number, o.order_date, o.content->'header'->>'title', o.project_id, o.created_at
	          FROM orders o`
	args := []interface{}{}
	if strings.TrimSpace(filter) != "" {
		query += ` WHERE o.order_number ILIKE $1 OR o.content->'header'->>'title' ILIKE $1`
		args = append(args, "%"+strings.TrimSpace(filter)+"%")
	}
	query += ` ORDER BY o.created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []models.OrderSummary{}
	for rows.Next() {
		var s models.OrderSummary
		var title *string
		var projectID *string
		var orderDate time.Time
		if err := rows.Scan(&s.ID, &s.OrderNumber, &orderDate, &title, &projectID, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.OrderDate = orderDate.Format("2006-01-02")
		if title != nil {
			s.Title = *title
		}
		if projectID != nil {
			s.ProjectID = *projectID
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *OrderRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Pool.Exec(ctx, `DELETE FROM orders WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("приказ не найден")
	}
	return nil
}

// NextNumber возвращает следующий порядковый номер приказа.
func (r *OrderRepository) NextNumber(ctx context.Context, prefix string) (string, error) {
	var cnt int
	if err := r.db.Pool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&cnt); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d", prefix, cnt+1), nil
}

func nullUUID(id string) interface{} {
	if id == "" {
		return nil
	}
	return id
}
