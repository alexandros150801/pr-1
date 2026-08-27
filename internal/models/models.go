package models

import "time"

// ---------- Структура документа (хранится в JSONB) ----------

// HeaderInfo — шапка документа.
type HeaderInfo struct {
	OrgName   string `json:"orgName"`   // наименование организации
	DocType   string `json:"docType"`   // вид документа (ПРИКАЗ и т.п.)
	DocNumber string `json:"docNumber"` // номер (для проекта может быть пустым)
	DocDate   string `json:"docDate"`   // дата в формате ГГГГ-ММ-ДД (может быть пустой)
	Place     string `json:"place"`     // место составления
	Title     string `json:"title"`     // заголовок (о чём приказ)
}

// OrderItem — пункт приказа (с поддержкой подпунктов).
type OrderItem struct {
	ID       string      `json:"id"`
	Text     string      `json:"text"`
	Children []OrderItem `json:"children"`
}

// Signature — подпись должностного лица.
type Signature struct {
	Position string `json:"position"`
	FullName string `json:"fullName"`
	Note     string `json:"note"`
}

// Attachment — приложение к приказу.
type Attachment struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	FileName    string `json:"fileName"`
}

// DocumentContent — полная структура приказа/проекта/выписки.
type DocumentContent struct {
	Header      HeaderInfo   `json:"header"`
	Preamble    string       `json:"preamble"`    // преамбула (основание)
	ActionWord  string       `json:"actionWord"`  // например "ПРИКАЗЫВАЮ"
	Items       []OrderItem  `json:"items"`       // пункты
	Signatures  []Signature  `json:"signatures"`  // подписи
	Attachments []Attachment `json:"attachments"` // приложения
	ControlNote string       `json:"controlNote"` // отметка о контроле
}

// ---------- Проекты ----------

type ProjectStatus string

const (
	ProjectStatusDraft    ProjectStatus = "draft"
	ProjectStatusApproved ProjectStatus = "approved"
)

type Project struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	Content   DocumentContent `json:"content"`
	Status    ProjectStatus   `json:"status"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type ProjectSummary struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Status    ProjectStatus `json:"status"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// ProjectDTO — данные для создания/обновления проекта.
type ProjectDTO struct {
	Title   string          `json:"title"`
	Content DocumentContent `json:"content"`
	Status  ProjectStatus   `json:"status"`
}

// ---------- Приказы ----------

type Order struct {
	ID          string          `json:"id"`
	ProjectID   string          `json:"projectId"`
	OrderNumber string          `json:"orderNumber"`
	OrderDate   string          `json:"orderDate"` // ГГГГ-ММ-ДД
	Content     DocumentContent `json:"content"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

type OrderSummary struct {
	ID          string    `json:"id"`
	OrderNumber string    `json:"orderNumber"`
	OrderDate   string    `json:"orderDate"`
	Title       string    `json:"title"`
	ProjectID   string    `json:"projectId"`
	CreatedAt   time.Time `json:"createdAt"`
}

// OrderDTO — данные для обновления приказа.
type OrderDTO struct {
	OrderNumber string          `json:"orderNumber"`
	OrderDate   string          `json:"orderDate"`
	Content     DocumentContent `json:"content"`
}

// ---------- Выписки ----------

type Extract struct {
	ID        string          `json:"id"`
	OrderID   string          `json:"orderId"`
	Title     string          `json:"title"`
	Content   DocumentContent `json:"content"`
	CreatedAt time.Time       `json:"createdAt"`
}

type ExtractSummary struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"orderId"`
	Title     string    `json:"title"`
	OrderInfo string    `json:"orderInfo"`
	CreatedAt time.Time `json:"createdAt"`
}

// ---------- Справочники ----------

type Employee struct {
	ID              string `json:"id"`
	FullName        string `json:"fullName"`
	Position        string `json:"position"`
	Department      string `json:"department"`
	PersonnelNumber string `json:"personnelNumber"`
}

type Department struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	HeadID       string `json:"headId"`
	HeadFullName string `json:"headFullName"`
}

type Base struct {
	ID        string `json:"id"`
	DocType   string `json:"docType"`
	DocNumber string `json:"docNumber"`
	DocDate   string `json:"docDate"`
	Summary   string `json:"summary"`
}

// ---------- Импорт Excel ----------

type ExcelPreview struct {
	Sheets    []string  `json:"sheets"`
	SheetName string    `json:"sheetName"`
	Headers   []string  `json:"headers"`
	Columns   []string  `json:"columns"` // буквы колонок (A, B, C...)
	Rows      [][]string `json:"rows"`
	TotalRows int       `json:"totalRows"`
}

// ImportMapping — настройки импорта.
type ImportMapping struct {
	TargetTable       string            `json:"targetTable"`       // employees | departments | bases
	SheetName         string            `json:"sheetName"`
	HeaderRowIndex    int               `json:"headerRowIndex"`    // строка заголовков (с 1)
	DataStartRowIndex int               `json:"dataStartRowIndex"` // первая строка данных (с 1)
	Columns           map[string]string `json:"columns"`           // поле таблицы -> буква колонки
	DuplicateMode     string            `json:"duplicateMode"`     // skip | update | create
}

type ImportRowError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

type ImportResult struct {
	Total    int              `json:"total"`
	Inserted int              `json:"inserted"`
	Updated  int              `json:"updated"`
	Skipped  int              `json:"skipped"`
	Errors   []ImportRowError `json:"errors"`
}

// ---------- Шаблоны ----------

// DocxStyle — параметры оформления DOCX.
type DocxStyle struct {
	FontName      string  `json:"fontName"`
	FontSizePt    int     `json:"fontSizePt"`
	LineSpacing   float64 `json:"lineSpacing"` // множитель, например 1.5
	ShowEmblem    bool    `json:"showEmblem"`
	MarginTopCm    float64 `json:"marginTopCm"`
	MarginBottomCm float64 `json:"marginBottomCm"`
	MarginLeftCm   float64 `json:"marginLeftCm"`
	MarginRightCm  float64 `json:"marginRightCm"`
}

type Template struct {
	ID          string          `json:"id"`
	Kind        string          `json:"kind"` // project | docx
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Content     DocumentContent `json:"content"` // для kind=project
	Style       DocxStyle       `json:"style"`   // для kind=docx
	CreatedAt   time.Time       `json:"createdAt"`
}

// ---------- Настройки ----------

type AppSettings struct {
	DefaultExportDir string `json:"defaultExportDir"`
	AutoNumberOrders bool   `json:"autoNumberOrders"`
	NumberPrefix     string `json:"numberPrefix"`
	OrgName          string `json:"orgName"`
	Place            string `json:"place"`
}

type ConfigInfo struct {
	DBHost     string `json:"dbHost"`
	DBPort     int    `json:"dbPort"`
	DBName     string `json:"dbName"`
	DBUser     string `json:"dbUser"`
	LogFile    string `json:"logFile"`
	ConfigFile string `json:"configFile"`
	Version    string `json:"version"`
}