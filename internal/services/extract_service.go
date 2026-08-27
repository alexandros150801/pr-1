package services

import (
	"context"
	"fmt"
	"strings"

	"pr-1/internal/models"
	"pr-1/internal/repositories"
	"pr-1/internal/utils"
)

// ExtractService — бизнес-логика выписок из приказов.
type ExtractService struct {
	extracts *repositories.ExtractRepository
	orders   *repositories.OrderRepository
	log      *utils.Logger
}

func NewExtractService(extracts *repositories.ExtractRepository, orders *repositories.OrderRepository, log *utils.Logger) *ExtractService {
	return &ExtractService{extracts: extracts, orders: orders, log: log}
}

// filterItems оставляет только пункты с указанными идентификаторами (включая подпункты).
func filterItems(items []models.OrderItem, selected map[string]bool) []models.OrderItem {
	result := []models.OrderItem{}
	for _, it := range items {
		if !selected[it.ID] {
			continue
		}
		copyItem := models.OrderItem{ID: it.ID, Text: it.Text}
		if len(it.Children) > 0 {
			copyItem.Children = filterItems(it.Children, selected)
		}
		result = append(result, copyItem)
	}
	return result
}

// CreateExtract формирует выписку из приказа по выбранным пунктам.
func (s *ExtractService) CreateExtract(ctx context.Context, orderID string, selectedItemIDs []string, title string) (*models.Extract, error) {
	order, err := s.orders.Get(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("приказ не найден: %w", err)
	}
	if len(selectedItemIDs) == 0 {
		return nil, fmt.Errorf("не выбраны пункты для выписки")
	}

	selected := make(map[string]bool, len(selectedItemIDs))
	for _, id := range selectedItemIDs {
		selected[id] = true
	}

	items := filterItems(order.Content.Items, selected)
	if len(items) == 0 {
		return nil, fmt.Errorf("выбранные пункты не найдены в приказе")
	}

	if strings.TrimSpace(title) == "" {
		title = fmt.Sprintf("Выписка из приказа № %s от %s", order.OrderNumber, formatDateRu(order.OrderDate))
	}

	content := models.DocumentContent{
		Header: models.HeaderInfo{
			OrgName:   order.Content.Header.OrgName,
			DocType:   "ВЫПИСКА ИЗ ПРИКАЗА",
			DocNumber: order.OrderNumber,
			DocDate:   order.OrderDate,
			Place:     order.Content.Header.Place,
			Title:     order.Content.Header.Title,
		},
		Preamble:   order.Content.Preamble,
		ActionWord: order.Content.ActionWord,
		Items:      items,
		Signatures: order.Content.Signatures,
	}

	extract, err := s.extracts.Create(ctx, orderID, title, content)
	if err != nil {
		s.log.Error("создание выписки из приказа %s: %v", orderID, err)
		return nil, err
	}
	s.log.Info("создана выписка %s из приказа %s", extract.ID, orderID)
	return extract, nil
}

func (s *ExtractService) Get(ctx context.Context, id string) (*models.Extract, error) {
	return s.extracts.Get(ctx, id)
}

func (s *ExtractService) List(ctx context.Context, filter string) ([]models.ExtractSummary, error) {
	return s.extracts.List(ctx, filter)
}

func (s *ExtractService) Delete(ctx context.Context, id string) error {
	if err := s.extracts.Delete(ctx, id); err != nil {
		return err
	}
	s.log.Info("удалена выписка %s", id)
	return nil
}

// formatDateRu преобразует ГГГГ-ММ-ДД в ДД.ММ.ГГГГ.
func formatDateRu(iso string) string {
	if len(iso) != 10 {
		return iso
	}
	return iso[8:10] + "." + iso[5:7] + "." + iso[0:4]
}
