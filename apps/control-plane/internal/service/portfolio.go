package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

var slugCharacters = regexp.MustCompile(`[^a-z0-9]+`)

type CreateAccountInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Service) CreateAccount(ctx context.Context, input CreateAccountInput) (domain.Account, error) {
	if strings.TrimSpace(input.Name) == "" {
		return domain.Account{}, fmt.Errorf("%w: account name is required", ErrInvalid)
	}
	if input.Slug == "" {
		input.Slug = slugify(input.Name)
	}
	account := domain.Account{ID: newID("acc"), Name: strings.TrimSpace(input.Name), Slug: slugify(input.Slug), CreatedAt: s.now()}
	if account.Slug == "" {
		return domain.Account{}, fmt.Errorf("%w: account slug is invalid", ErrInvalid)
	}
	if err := s.repo.CreateAccount(ctx, account); err != nil {
		return domain.Account{}, err
	}
	return account, nil
}

type CreateProjectInput struct {
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Repository string `json:"repository"`
	Owner      string `json:"owner"`
}

func (s *Service) CreateProject(ctx context.Context, accountID string, input CreateProjectInput) (domain.Project, error) {
	if strings.TrimSpace(input.Name) == "" {
		return domain.Project{}, fmt.Errorf("%w: project name is required", ErrInvalid)
	}
	if _, err := s.repo.GetAccount(ctx, accountID); err != nil {
		return domain.Project{}, err
	}
	if input.Slug == "" {
		input.Slug = input.Name
	}
	project := domain.Project{
		ID: newID("prj"), AccountID: accountID, Name: strings.TrimSpace(input.Name), Slug: slugify(input.Slug),
		Repository: strings.TrimSpace(input.Repository), Owner: strings.TrimSpace(input.Owner), CreatedAt: s.now(),
	}
	if project.Slug == "" {
		return domain.Project{}, fmt.Errorf("%w: project slug is invalid", ErrInvalid)
	}
	if err := s.repo.CreateProject(ctx, project); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

type CreateAISystemInput struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Model       string   `json:"model"`
	Purpose     string   `json:"purpose"`
	DataClasses []string `json:"data_classes"`
	Tools       []string `json:"tools"`
	Owner       string   `json:"owner"`
}

func (s *Service) CreateAISystem(ctx context.Context, projectID string, input CreateAISystemInput) (domain.AISystem, error) {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Provider) == "" || strings.TrimSpace(input.Model) == "" || strings.TrimSpace(input.Purpose) == "" {
		return domain.AISystem{}, fmt.Errorf("%w: AI system name, provider, model, and purpose are required", ErrInvalid)
	}
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return domain.AISystem{}, err
	}
	now := s.now()
	system := domain.AISystem{
		ID: newID("ais"), AccountID: project.AccountID, ProjectID: projectID, Name: strings.TrimSpace(input.Name),
		Provider: strings.TrimSpace(input.Provider), Model: strings.TrimSpace(input.Model), Purpose: strings.TrimSpace(input.Purpose),
		DataClasses: nonNil(input.DataClasses), Tools: nonNil(input.Tools), Owner: strings.TrimSpace(input.Owner),
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateAISystem(ctx, system); err != nil {
		return domain.AISystem{}, err
	}
	return system, nil
}

func (s *Service) AccountDashboard(ctx context.Context, accountID string) (domain.AccountDashboard, error) {
	return s.repo.GetAccountDashboard(ctx, accountID, s.now())
}

func slugify(value string) string {
	value = strings.Trim(slugCharacters.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "-"), "-")
	return value
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
