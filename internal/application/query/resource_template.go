package query

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed resource_templates.json
var resourceTemplatesJSON []byte

type ResourceTemplatePort struct {
	Host      string `json:"host"`
	Container string `json:"container"`
}

type ResourceTemplateEnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResourceTemplate struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	IconURL     string                   `json:"icon_url"`
	Image       string                   `json:"image"`
	Description string                   `json:"description"`
	Color       string                   `json:"color"`
	Abbr        string                   `json:"abbr"`
	Tags        []string                 `json:"tags"`
	Ports       []ResourceTemplatePort   `json:"ports"`
	Env         []ResourceTemplateEnvVar `json:"env"`
	Type        string                   `json:"type"`
	Volumes     []string                 `json:"volumes,omitempty"`
	Cmd         []string                 `json:"cmd,omitempty"`
}

type ListResourceTemplatesQuery struct{}

type ListResourceTemplatesHandler struct {
	templates []ResourceTemplate
}

func NewListResourceTemplatesHandler() (*ListResourceTemplatesHandler, error) {
	var templates []ResourceTemplate
	if err := json.Unmarshal(resourceTemplatesJSON, &templates); err != nil {
		return nil, fmt.Errorf("decode resource templates: %w", err)
	}
	return &ListResourceTemplatesHandler{templates: templates}, nil
}

func (h *ListResourceTemplatesHandler) Handle(_ context.Context, _ ListResourceTemplatesQuery) ([]ResourceTemplate, error) {
	out := make([]ResourceTemplate, len(h.templates))
	copy(out, h.templates)
	return out, nil
}
