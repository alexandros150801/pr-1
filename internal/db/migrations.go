package db

import (
	"context"
	"encoding/json"
	"fmt"

	"pr-1/internal/models"
)

const schemaSQL = `
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    content JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    order_number VARCHAR(50) NOT NULL,
    order_date DATE NOT NULL,
    content JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'issued',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT orders_number_unique UNIQUE (order_number)
);

CREATE TABLE IF NOT EXISTS extracts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    title VARCHAR(255),
    content JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name VARCHAR(255),
    position VARCHAR(255),
    department VARCHAR(255),
    personnel_number VARCHAR(50),
    additional_data JSONB DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    code VARCHAR(50),
    head_employee_id UUID REFERENCES employees(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    doc_type VARCHAR(100),
    doc_number VARCHAR(50),
    doc_date DATE,
    summary TEXT
);

CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind VARCHAR(20) NOT NULL, -- project | docx
    name VARCHAR(255) NOT NULL,
    description TEXT,
    content JSONB,
    style JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_orders_project ON orders(project_id);
CREATE INDEX IF NOT EXISTS idx_extracts_order ON extracts(order_id);
CREATE INDEX IF NOT EXISTS idx_employees_name ON employees(full_name);
CREATE INDEX IF NOT EXISTS idx_departments_name ON departments(name);
`

// Migrate применяет схему БД и заполняет начальные данные.
func (d *DB) Migrate(ctx context.Context) error {
	if _, err := d.Pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("ошибка применения схемы БД: %w", err)
	}
	if err := d.seed(ctx); err != nil {
		return fmt.Errorf("ошибка заполнения начальных данных: %w", err)
	}
	d.Log.Info("миграции БД применены")
	return nil
}

// seed вставляет начальные шаблоны и настройки, если их ещё нет.
func (d *DB) seed(ctx context.Context) error {
	// шаблон проекта по умолчанию
	var cnt int
	if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM templates WHERE kind='project'`).Scan(&cnt); err != nil {
		return err
	}
	if cnt == 0 {
		def := models.DocumentContent{
			Header: models.HeaderInfo{
				OrgName: "Наименование организации",
				DocType: "ПРИКАЗ",
				Place:   "г. Москва",
			},
			Preamble:   "В соответствии с ...",
			ActionWord: "ПРИКАЗЫВАЮ:",
			Items: []models.OrderItem{
				{ID: "item-1", Text: "Текст пункта приказа"},
			},
			Signatures: []models.Signature{
				{Position: "Директор", FullName: "И.О. Фамилия"},
			},
		}
		b, _ := json.Marshal(def)
		if _, err := d.Pool.Exec(ctx,
			`INSERT INTO templates (kind, name, description, content) VALUES ('project', $1, $2, $3)`,
			"Стандартный шаблон приказа", "Базовый шаблон проекта приказа", b); err != nil {
			return err
		}
	}

	// шаблоны оформления DOCX
	if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM templates WHERE kind='docx'`).Scan(&cnt); err != nil {
		return err
	}
	if cnt == 0 {
		std := models.DocxStyle{FontName: "Times New Roman", FontSizePt: 12, LineSpacing: 1.5,
			MarginTopCm: 2, MarginBottomCm: 2, MarginLeftCm: 3, MarginRightCm: 1.5}
		emblem := std
		emblem.ShowEmblem = true
		b1, _ := json.Marshal(std)
		b2, _ := json.Marshal(emblem)
		if _, err := d.Pool.Exec(ctx,
			`INSERT INTO templates (kind, name, description, style) VALUES ('docx', $1, $2, $3), ('docx', $4, $5, $6)`,
			"Стандартный", "Оформление по ГОСТ: Times New Roman 12пт, полуторный интервал", b1,
			"С гербом", "Стандартное оформление с местом для герба", b2); err != nil {
			return err
		}
	}

	// настройки по умолчанию
	if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM settings WHERE key='app'`).Scan(&cnt); err != nil {
		return err
	}
	if cnt == 0 {
		app := models.AppSettings{
			AutoNumberOrders: true,
			NumberPrefix:     "",
			OrgName:          "Наименование организации",
			Place:            "г. Москва",
		}
		b, _ := json.Marshal(app)
		if _, err := d.Pool.Exec(ctx, `INSERT INTO settings (key, value) VALUES ('app', $1)`, b); err != nil {
			return err
		}
	}
	return nil
}
