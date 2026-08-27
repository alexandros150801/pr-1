package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"pr-1/internal/models"
	"pr-1/internal/repositories"
	"pr-1/internal/utils"
)

// OrderService — бизнес-логика приказов.
type OrderService struct {
	orders   *repositories.OrderRepository
	projects *repositories.ProjectRepository
	settings *repositories.SettingsRepository
	log      *utils.Logger
}

func NewOrderService(orders *repositories.OrderRepository, projects *repositories.ProjectRepository, settings *repositories.SettingsRepository, log *utils.Logger) *OrderService {
	return &OrderService{orders: orders, projects: projects, settings: settings, log: log}
}

func parseDate(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("не указана дата приказа")
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return fmt.Errorf("неверный формат даты (ожидается ГГГГ-ММ-ДД)")
	}
	return nil
}

// GenerateOrder создаёт приказ на основе проекта.
func (s *OrderService) GenerateOrder(ctx context.Context, projectID string, dto models.OrderDTO) (*models.Order, error) {
	project, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("проект для формирования приказа не найден: %w", err)
	}

	number := strings.TrimSpace(dto.OrderNumber)
	date := strings.TrimSpace(dto.OrderDate)

	// автоприсвоение номера при необходимости
	if number == "" {
		app, serr := s.settings.GetAppSettings(ctx)
		if serr != nil {
			return nil, serr
		}
		if app.AutoNumberOrders {
			number, err = s.orders.NextNumber(ctx, app.NumberPrefix)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("не указан регистрационный номер приказа")
		}
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if err := parseDate(date); err != nil {
		return nil, err
	}

	// содержимое приказа: из DTO либо из проекта с подстановкой номера/даты в шапку
	content := dto.Content
	if len(content.Items) == 0 && len(content.Header.DocType) == 0 && content.Preamble == "" {
		content = project.Content
	}
	content.Header.DocNumber = number
	content.Header.DocDate = date
	if content.Header.DocType == "" {
		content.Header.DocType = "ПРИКАЗ"
	}

	order, err := s.orders.Create(ctx, projectID, number, date, content)
	if err != nil {
		s.log.Error("формирование приказа из проекта %s: %v", projectID, err)
		return nil, err
	}

	// проект переводится в статус "утверждён" и получает номер в шапке
	project.Status = models.ProjectStatusApproved
	project.Content.Header.DocNumber = number
	project.Content.Header.DocDate = date
	if _, err := s.projects.Update(ctx, projectID, models.ProjectDTO{
		Title:   project.Title,
		Content: project.Content,
		Status:  project.Status,
	}); err != nil {
		s.log.Warn("не удалось обновить проект после формирования приказа: %v", err)
	}

	s.log.Info("сформирован приказ %s №%s от %s", order.ID, number, date)
	return order, nil
}

func (s *OrderService) Update(ctx context.Context, id string, dto models.OrderDTO) (*models.Order, error) {
	if strings.TrimSpace(dto.OrderNumber) == "" {
		return nil, fmt.Errorf("не указан номер приказа")
	}
	if err := parseDate(dto.OrderDate); err != nil {
		return nil, err
	}
	dto.Content.Header.DocNumber = dto.OrderNumber
	dto.Content.Header.DocDate = dto.OrderDate
	return s.orders.Update(ctx, id, dto)
}

func (s *OrderService) Get(ctx context.Context, id string) (*models.Order, error) {
	return s.orders.Get(ctx, id)
}

func (s *OrderService) List(ctx context.Context, filter string) ([]models.OrderSummary, error) {
	return s.orders.List(ctx, filter)
}

func (s *OrderService) Delete(ctx context.Context, id string) error {
	if err := s.orders.Delete(ctx, id); err != nil {
		return err
	}
	s.log.Info("удалён приказ %s", id)
	return nil
}

// NextNumber предлагает следующий номер приказа.
func (s *OrderService) NextNumber(ctx context.Context) (string, error) {
	app, err := s.settings.GetAppSettings(ctx)
	if err != nil {
		return "", err
	}
	return s.orders.NextNumber(ctx, app.NumberPrefix)
}
