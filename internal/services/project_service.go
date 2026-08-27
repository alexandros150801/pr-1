package services

import (
	"context"
	"fmt"
	"strings"

	"pr-1/internal/models"
	"pr-1/internal/repositories"
	"pr-1/internal/utils"
)

// ProjectService — бизнес-логика проектов приказов.
type ProjectService struct {
	repo *repositories.ProjectRepository
	log  *utils.Logger
}

func NewProjectService(repo *repositories.ProjectRepository, log *utils.Logger) *ProjectService {
	return &ProjectService{repo: repo, log: log}
}

func (s *ProjectService) validate(dto models.ProjectDTO) error {
	if strings.TrimSpace(dto.Title) == "" {
		return fmt.Errorf("не указано название проекта")
	}
	if len(dto.Title) > 255 {
		return fmt.Errorf("название проекта слишком длинное (максимум 255 символов)")
	}
	if dto.Status != "" && dto.Status != models.ProjectStatusDraft && dto.Status != models.ProjectStatusApproved {
		return fmt.Errorf("недопустимый статус проекта: %s", dto.Status)
	}
	return nil
}

func (s *ProjectService) Create(ctx context.Context, dto models.ProjectDTO) (*models.Project, error) {
	if err := s.validate(dto); err != nil {
		return nil, err
	}
	p, err := s.repo.Create(ctx, dto)
	if err != nil {
		s.log.Error("создание проекта: %v", err)
		return nil, err
	}
	s.log.Info("создан проект %s (%s)", p.ID, p.Title)
	return p, nil
}

func (s *ProjectService) Update(ctx context.Context, id string, dto models.ProjectDTO) (*models.Project, error) {
	if err := s.validate(dto); err != nil {
		return nil, err
	}
	p, err := s.repo.Update(ctx, id, dto)
	if err != nil {
		s.log.Error("обновление проекта %s: %v", id, err)
		return nil, err
	}
	return p, nil
}

func (s *ProjectService) Get(ctx context.Context, id string) (*models.Project, error) {
	return s.repo.Get(ctx, id)
}

func (s *ProjectService) List(ctx context.Context, filter string) ([]models.ProjectSummary, error) {
	return s.repo.List(ctx, filter)
}

func (s *ProjectService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.log.Info("удалён проект %s", id)
	return nil
}
