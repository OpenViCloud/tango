package query

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"tango/internal/domain"
)

//go:embed resource_stack_templates.json
var resourceStackTemplatesJSON []byte

type ListResourceStackTemplatesQuery struct{}

type ListResourceStackTemplatesHandler struct {
	templates []domain.ResourceStackTemplate
}

func LoadResourceStackTemplates() ([]domain.ResourceStackTemplate, error) {
	var templates []domain.ResourceStackTemplate
	if err := json.Unmarshal(resourceStackTemplatesJSON, &templates); err != nil {
		return nil, fmt.Errorf("decode resource stack templates: %w", err)
	}
	return templates, nil
}

func NewListResourceStackTemplatesHandler() (*ListResourceStackTemplatesHandler, error) {
	templates, err := LoadResourceStackTemplates()
	if err != nil {
		return nil, err
	}
	return &ListResourceStackTemplatesHandler{templates: templates}, nil
}

func (h *ListResourceStackTemplatesHandler) Handle(_ context.Context, _ ListResourceStackTemplatesQuery) ([]domain.ResourceStackTemplate, error) {
	out := make([]domain.ResourceStackTemplate, len(h.templates))
	copy(out, h.templates)
	return out, nil
}
