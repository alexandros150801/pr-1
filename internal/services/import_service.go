package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/xuri/excelize/v2"

	"pr-1/internal/db"
	"pr-1/internal/models"
	"pr-1/internal/repositories"
	"pr-1/internal/utils"
)

// ImportService — импорт данных из Excel.
type ImportService struct {
	database *db.DB
	refs     *repositories.ReferenceRepository
	log      *utils.Logger
}

func NewImportService(database *db.DB, refs *repositories.ReferenceRepository, log *utils.Logger) *ImportService {
	return &ImportService{database: database, refs: refs, log: log}
}

// PreviewExcel читает файл и возвращает предпросмотр данных.
func (s *ImportService) PreviewExcel(filePath string, previewRows int) (*models.ExcelPreview, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл Excel: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("в файле нет листов")
	}
	sheet := sheets[0]

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать лист: %w", err)
	}

	preview := &models.ExcelPreview{
		Sheets:    sheets,
		SheetName: sheet,
		TotalRows: len(rows),
	}

	if previewRows <= 0 {
		previewRows = 10
	}

	// определяем максимальное число колонок по первым строкам
	maxCols := 0
	limit := previewRows + 1
	if limit > len(rows) {
		limit = len(rows)
	}
	for i := 0; i < limit; i++ {
		if len(rows[i]) > maxCols {
			maxCols = len(rows[i])
		}
	}
	// заголовки колонок — буквы A..Z, AA..
	for i := 0; i < maxCols; i++ {
		col, _ := excelize.ColumnNumberToName(i + 1)
		preview.Columns = append(preview.Columns, col)
	}

	for i := 0; i < limit; i++ {
		row := make([]string, maxCols)
		for j := 0; j < maxCols && j < len(rows[i]); j++ {
			row[j] = rows[i][j]
		}
		if i == 0 {
			preview.Headers = row
		} else {
			preview.Rows = append(preview.Rows, row)
		}
	}
	if preview.Rows == nil {
		preview.Rows = [][]string{}
	}
	return preview, nil
}

// ImportExcel выполняет импорт данных в целевую таблицу в транзакции.
func (s *ImportService) ImportExcel(ctx context.Context, filePath string, mapping models.ImportMapping) (*models.ImportResult, error) {
	switch mapping.TargetTable {
	case "employees", "departments", "bases":
	default:
		return nil, fmt.Errorf("недопустимая целевая таблица: %s", mapping.TargetTable)
	}
	if mapping.DuplicateMode == "" {
		mapping.DuplicateMode = "create"
	}
	switch mapping.DuplicateMode {
	case "skip", "update", "create":
	default:
		return nil, fmt.Errorf("недопустимый режим обработки дубликатов: %s", mapping.DuplicateMode)
	}

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл Excel: %w", err)
	}
	defer f.Close()

	sheet := mapping.SheetName
	if sheet == "" {
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("в файле нет листов")
		}
		sheet = sheets[0]
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать лист %s: %w", sheet, err)
	}

	startIdx := mapping.DataStartRowIndex - 1
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx >= len(rows) {
		return &models.ImportResult{Errors: []models.ImportRowError{}}, nil
	}

	tx, err := s.database.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer tx.Rollback(ctx)

	result := &models.ImportResult{Errors: []models.ImportRowError{}}

	for i := startIdx; i < len(rows); i++ {
		row := rows[i]
		rowNum := i + 1 // номер строки в файле (с 1)

		values := make(map[string]string)
		hasData := false
		for field, col := range mapping.Columns {
			if col == "" {
				continue
			}
			colIdx, cerr := excelize.ColumnNameToNumber(col)
			if cerr != nil {
				return nil, fmt.Errorf("неверная колонка %s для поля %s", col, field)
			}
			val := ""
			if colIdx-1 < len(row) {
				val = strings.TrimSpace(row[colIdx-1])
			}
			values[field] = val
			if val != "" {
				hasData = true
			}
		}
		if !hasData {
			result.Skipped++
			continue
		}

		var rerr error
		switch mapping.TargetTable {
		case "employees":
			rerr = s.importEmployee(ctx, tx, values, mapping.DuplicateMode, result)
		case "departments":
			rerr = s.importDepartment(ctx, tx, values, mapping.DuplicateMode, result)
		case "bases":
			rerr = s.importBase(ctx, tx, values, mapping.DuplicateMode, result)
		}
		if rerr != nil {
			result.Errors = append(result.Errors, models.ImportRowError{Row: rowNum, Message: rerr.Error()})
		}
		result.Total++
	}

	if len(result.Errors) > 0 && result.Inserted == 0 && result.Updated == 0 {
		return result, fmt.Errorf("импорт не выполнен: все строки содержат ошибки")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("ошибка фиксации транзакции: %w", err)
	}

	s.log.Info("импорт завершён: таблица=%s, добавлено=%d, обновлено=%d, пропущено=%d, ошибок=%d",
		mapping.TargetTable, result.Inserted, result.Updated, result.Skipped, len(result.Errors))
	return result, nil
}

func (s *ImportService) importEmployee(ctx context.Context, tx pgx.Tx, values map[string]string, mode string, result *models.ImportResult) error {
	e := models.Employee{
		FullName:        values["full_name"],
		Position:        values["position"],
		Department:      values["department"],
		PersonnelNumber: values["personnel_number"],
	}
	if e.FullName == "" {
		result.Skipped++
		return nil
	}

	// поиск дубликата по табельному номеру либо по ФИО
	var existing *models.Employee
	var err error
	if e.PersonnelNumber != "" {
		existing, err = s.findEmployeeByNumberTx(ctx, tx, e.PersonnelNumber)
	} else {
		existing, err = s.findEmployeeByNameTx(ctx, tx, e.FullName)
	}
	if err != nil {
		return err
	}

	if existing != nil {
		switch mode {
		case "skip":
			result.Skipped++
			return nil
		case "update":
			if _, err := tx.Exec(ctx,
				`UPDATE employees SET full_name=$2, position=$3, department=$4, personnel_number=$5 WHERE id=$1`,
				existing.ID, e.FullName, e.Position, e.Department, e.PersonnelNumber); err != nil {
				return err
			}
			result.Updated++
			return nil
		}
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO employees (full_name, position, department, personnel_number) VALUES ($1, $2, $3, $4)`,
		e.FullName, e.Position, e.Department, e.PersonnelNumber); err != nil {
		return err
	}
	result.Inserted++
	return nil
}

func (s *ImportService) importDepartment(ctx context.Context, tx pgx.Tx, values map[string]string, mode string, result *models.ImportResult) error {
	d := models.Department{
		Name: values["name"],
		Code: values["code"],
	}
	if d.Name == "" {
		result.Skipped++
		return nil
	}

	var existing *models.Department
	var err error
	if d.Code != "" {
		existing, err = s.findDepartmentByCodeTx(ctx, tx, d.Code)
	} else {
		existing, err = s.findDepartmentByNameTx(ctx, tx, d.Name)
	}
	if err != nil {
		return err
	}

	if existing != nil {
		switch mode {
		case "skip":
			result.Skipped++
			return nil
		case "update":
			if _, err := tx.Exec(ctx, `UPDATE departments SET name=$2, code=$3 WHERE id=$1`,
				existing.ID, d.Name, d.Code); err != nil {
				return err
			}
			result.Updated++
			return nil
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO departments (name, code) VALUES ($1, $2)`, d.Name, d.Code); err != nil {
		return err
	}
	result.Inserted++
	return nil
}

func (s *ImportService) importBase(ctx context.Context, tx pgx.Tx, values map[string]string, mode string, result *models.ImportResult) error {
	b := models.Base{
		DocType:   values["doc_type"],
		DocNumber: values["doc_number"],
		DocDate:   values["doc_date"],
		Summary:   values["summary"],
	}
	if b.DocType == "" && b.DocNumber == "" && b.Summary == "" {
		result.Skipped++
		return nil
	}

	var existing *models.Base
	if b.DocNumber != "" {
		var err error
		existing, err = s.findBaseByNumberTx(ctx, tx, b.DocNumber)
		if err != nil {
			return err
		}
	}

	if existing != nil {
		switch mode {
		case "skip":
			result.Skipped++
			return nil
		case "update":
			if _, err := tx.Exec(ctx,
				`UPDATE bases SET doc_type=$2, doc_number=$3, doc_date=NULLIF($4,'')::date, summary=$5 WHERE id=$1`,
				existing.ID, b.DocType, b.DocNumber, b.DocDate, b.Summary); err != nil {
				return err
			}
			result.Updated++
			return nil
		}
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO bases (doc_type, doc_number, doc_date, summary) VALUES ($1, $2, NULLIF($3,'')::date, $4)`,
		b.DocType, b.DocNumber, b.DocDate, b.Summary); err != nil {
		return err
	}
	result.Inserted++
	return nil
}

// ---------- поиск в транзакции ----------

func (s *ImportService) findEmployeeByNumberTx(ctx context.Context, tx pgx.Tx, number string) (*models.Employee, error) {
	var e models.Employee
	err := tx.QueryRow(ctx, `SELECT id, coalesce(full_name,''), coalesce(position,''), coalesce(department,''), coalesce(personnel_number,'')
		FROM employees WHERE personnel_number=$1 LIMIT 1`, number).
		Scan(&e.ID, &e.FullName, &e.Position, &e.Department, &e.PersonnelNumber)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *ImportService) findEmployeeByNameTx(ctx context.Context, tx pgx.Tx, name string) (*models.Employee, error) {
	var e models.Employee
	err := tx.QueryRow(ctx, `SELECT id, coalesce(full_name,''), coalesce(position,''), coalesce(department,''), coalesce(personnel_number,'')
		FROM employees WHERE full_name=$1 LIMIT 1`, name).
		Scan(&e.ID, &e.FullName, &e.Position, &e.Department, &e.PersonnelNumber)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *ImportService) findDepartmentByCodeTx(ctx context.Context, tx pgx.Tx, code string) (*models.Department, error) {
	var d models.Department
	err := tx.QueryRow(ctx, `SELECT id, coalesce(name,''), coalesce(code,'') FROM departments WHERE code=$1 LIMIT 1`, code).
		Scan(&d.ID, &d.Name, &d.Code)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *ImportService) findDepartmentByNameTx(ctx context.Context, tx pgx.Tx, name string) (*models.Department, error) {
	var d models.Department
	err := tx.QueryRow(ctx, `SELECT id, coalesce(name,''), coalesce(code,'') FROM departments WHERE name=$1 LIMIT 1`, name).
		Scan(&d.ID, &d.Name, &d.Code)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *ImportService) findBaseByNumberTx(ctx context.Context, tx pgx.Tx, number string) (*models.Base, error) {
	var b models.Base
	err := tx.QueryRow(ctx, `SELECT id, coalesce(doc_type,''), coalesce(doc_number,''),
		coalesce(to_char(doc_date,'YYYY-MM-DD'),''), coalesce(summary,'')
		FROM bases WHERE doc_number=$1 LIMIT 1`, number).
		Scan(&b.ID, &b.DocType, &b.DocNumber, &b.DocDate, &b.Summary)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}
