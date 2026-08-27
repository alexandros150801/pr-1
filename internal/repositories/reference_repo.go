package repositories

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"pr-1/internal/db"
	"pr-1/internal/models"
)

// ReferenceRepository — доступ к справочным данным (сотрудники, подразделения, основания).
type ReferenceRepository struct {
	db *db.DB
}

func NewReferenceRepository(database *db.DB) *ReferenceRepository {
	return &ReferenceRepository{db: database}
}

// ---------- Сотрудники ----------

func (r *ReferenceRepository) ListEmployees(ctx context.Context, filter string) ([]models.Employee, error) {
	query := `SELECT id, coalesce(full_name,''), coalesce(position,''), coalesce(department,''), coalesce(personnel_number,'') FROM employees`
	args := []interface{}{}
	if strings.TrimSpace(filter) != "" {
		query += ` WHERE full_name ILIKE $1 OR position ILIKE $1 OR department ILIKE $1 OR personnel_number ILIKE $1`
		args = append(args, "%"+strings.TrimSpace(filter)+"%")
	}
	query += ` ORDER BY full_name`
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.Employee{}
	for rows.Next() {
		var e models.Employee
		if err := rows.Scan(&e.ID, &e.FullName, &e.Position, &e.Department, &e.PersonnelNumber); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *ReferenceRepository) CreateEmployee(ctx context.Context, e models.Employee) (*models.Employee, error) {
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO employees (full_name, position, department, personnel_number)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		e.FullName, e.Position, e.Department, e.PersonnelNumber).Scan(&e.ID)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *ReferenceRepository) UpdateEmployee(ctx context.Context, e models.Employee) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE employees SET full_name=$2, position=$3, department=$4, personnel_number=$5 WHERE id=$1`,
		e.ID, e.FullName, e.Position, e.Department, e.PersonnelNumber)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("сотрудник не найден")
	}
	return nil
}

func (r *ReferenceRepository) DeleteEmployee(ctx context.Context, id string) error {
	tag, err := r.db.Pool.Exec(ctx, `DELETE FROM employees WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("сотрудник не найден")
	}
	return nil
}

// FindEmployeeByNumber ищет сотрудника по табельному номеру.
func (r *ReferenceRepository) FindEmployeeByNumber(ctx context.Context, number string) (*models.Employee, error) {
	var e models.Employee
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, coalesce(full_name,''), coalesce(position,''), coalesce(department,''), coalesce(personnel_number,'')
		 FROM employees WHERE personnel_number=$1 LIMIT 1`, number).
		Scan(&e.ID, &e.FullName, &e.Position, &e.Department, &e.PersonnelNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ---------- Подразделения ----------

func (r *ReferenceRepository) ListDepartments(ctx context.Context, filter string) ([]models.Department, error) {
	query := `SELECT d.id, coalesce(d.name,''), coalesce(d.code,''),
	                 coalesce(d.head_employee_id::text, ''), coalesce(e.full_name, '')
	          FROM departments d LEFT JOIN employees e ON e.id = d.head_employee_id`
	args := []interface{}{}
	if strings.TrimSpace(filter) != "" {
		query += ` WHERE d.name ILIKE $1 OR d.code ILIKE $1`
		args = append(args, "%"+strings.TrimSpace(filter)+"%")
	}
	query += ` ORDER BY d.name`
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.Department{}
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.Code, &d.HeadID, &d.HeadFullName); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *ReferenceRepository) CreateDepartment(ctx context.Context, d models.Department) (*models.Department, error) {
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO departments (name, code, head_employee_id) VALUES ($1, $2, $3) RETURNING id`,
		d.Name, d.Code, nullUUID(d.HeadID)).Scan(&d.ID)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *ReferenceRepository) UpdateDepartment(ctx context.Context, d models.Department) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE departments SET name=$2, code=$3, head_employee_id=$4 WHERE id=$1`,
		d.ID, d.Name, d.Code, nullUUID(d.HeadID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("подразделение не найдено")
	}
	return nil
}

func (r *ReferenceRepository) DeleteDepartment(ctx context.Context, id string) error {
	tag, err := r.db.Pool.Exec(ctx, `DELETE FROM departments WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("подразделение не найдено")
	}
	return nil
}

// FindDepartmentByCode ищет подразделение по коду.
func (r *ReferenceRepository) FindDepartmentByCode(ctx context.Context, code string) (*models.Department, error) {
	var d models.Department
	var headID *string
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, coalesce(name,''), coalesce(code,''), head_employee_id::text
		 FROM departments WHERE code=$1 LIMIT 1`, code).
		Scan(&d.ID, &d.Name, &d.Code, &headID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if headID != nil {
		d.HeadID = *headID
	}
	return &d, nil
}

// ---------- Основания ----------

func (r *ReferenceRepository) ListBases(ctx context.Context, filter string) ([]models.Base, error) {
	query := `SELECT id, coalesce(doc_type,''), coalesce(doc_number,''),
	                 coalesce(to_char(doc_date, 'YYYY-MM-DD'), ''), coalesce(summary, '')
	          FROM bases`
	args := []interface{}{}
	if strings.TrimSpace(filter) != "" {
		query += ` WHERE doc_type ILIKE $1 OR doc_number ILIKE $1 OR summary ILIKE $1`
		args = append(args, "%"+strings.TrimSpace(filter)+"%")
	}
	query += ` ORDER BY doc_date DESC NULLS LAST`
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.Base{}
	for rows.Next() {
		var b models.Base
		if err := rows.Scan(&b.ID, &b.DocType, &b.DocNumber, &b.DocDate, &b.Summary); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

func (r *ReferenceRepository) CreateBase(ctx context.Context, b models.Base) (*models.Base, error) {
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO bases (doc_type, doc_number, doc_date, summary)
		 VALUES ($1, $2, NULLIF($3,'')::date, $4) RETURNING id`,
		b.DocType, b.DocNumber, b.DocDate, b.Summary).Scan(&b.ID)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *ReferenceRepository) UpdateBase(ctx context.Context, b models.Base) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE bases SET doc_type=$2, doc_number=$3, doc_date=NULLIF($4,'')::date, summary=$5 WHERE id=$1`,
		b.ID, b.DocType, b.DocNumber, b.DocDate, b.Summary)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("основание не найдено")
	}
	return nil
}

func (r *ReferenceRepository) DeleteBase(ctx context.Context, id string) error {
	tag, err := r.db.Pool.Exec(ctx, `DELETE FROM bases WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("основание не найдено")
	}
	return nil
}

// FindBaseByNumberDate ищет основание по номеру и дате.
func (r *ReferenceRepository) FindBaseByNumberDate(ctx context.Context, number, date string) (*models.Base, error) {
	var b models.Base
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, coalesce(doc_type,''), coalesce(doc_number,''),
		        coalesce(to_char(doc_date, 'YYYY-MM-DD'), ''), coalesce(summary, '')
		 FROM bases WHERE doc_number=$1 AND (NULLIF($2,'')::date IS NULL OR doc_date = NULLIF($2,'')::date)
		 LIMIT 1`, number, date).
		Scan(&b.ID, &b.DocType, &b.DocNumber, &b.DocDate, &b.Summary)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}
