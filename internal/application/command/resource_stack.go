package command

import (
	"context"
	"fmt"
	"slices"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type CustomComponentInput struct {
	ID      string
	Type    string // "service" | "job"
	Cmd     []string
	Ports   []ResourcePortInput
	Volumes []string
	Env     []ResourceEnvVarInput
}

type CreateResourceStackCommand struct {
	TemplateID       string
	NamePrefix       string
	Image            string
	Tag              string
	EnvironmentID    string
	CreatedBy        string
	NodeID           *string
	SharedEnvVars    []ResourceEnvVarInput
	CustomComponents []CustomComponentInput
}

type CreateResourceStackResult struct {
	TemplateID string
	Resources  []*domain.Resource
}

type CreateResourceStackHandler struct {
	createResource *CreateResourceHandler
	templates      map[string]domain.ResourceStackTemplate
	jobRunner      appservices.JobRunner
}

func NewCreateResourceStackHandler(
	createResource *CreateResourceHandler,
	templates []domain.ResourceStackTemplate,
	jobRunner appservices.JobRunner,
) *CreateResourceStackHandler {
	index := make(map[string]domain.ResourceStackTemplate, len(templates))
	for _, template := range templates {
		index[strings.TrimSpace(template.ID)] = template
	}
	return &CreateResourceStackHandler{
		createResource: createResource,
		templates:      index,
		jobRunner:      jobRunner,
	}
}

func (h *CreateResourceStackHandler) Handle(ctx context.Context, cmd CreateResourceStackCommand) (*CreateResourceStackResult, error) {
	template, ok := h.templates[strings.TrimSpace(cmd.TemplateID)]
	if !ok {
		return nil, domain.NewUserFacingError("Unknown resource stack template")
	}

	namePrefix := strings.TrimSpace(cmd.NamePrefix)
	if namePrefix == "" {
		namePrefix = template.ID
	}

	image := strings.TrimSpace(cmd.Image)
	if image == "" {
		image = strings.TrimSpace(template.Image)
	}
	if image == "" {
		return nil, domain.NewUserFacingError("Stack image is required")
	}

	tag := strings.TrimSpace(cmd.Tag)
	if tag == "" && len(template.Tags) > 0 {
		tag = strings.TrimSpace(template.Tags[0])
	}
	if tag == "" {
		tag = "latest"
	}

	sharedEnv := mergeEnvEntries(template.SharedEnv, cmd.SharedEnvVars)

	// Helper: create one resource record from a CustomComponentInput.
	createComp := func(comp CustomComponentInput, compType string) (*domain.Resource, error) {
		id := strings.TrimSpace(comp.ID)
		if id == "" {
			return nil, nil
		}
		config := map[string]any{}
		if len(comp.Cmd) > 0 {
			config["cmd"] = comp.Cmd
		}
		if len(comp.Volumes) > 0 {
			config["volumes"] = comp.Volumes
		}
		if len(config) == 0 {
			config = nil
		}
		envVars := mergeEnvVarInputs(sharedEnv, comp.Env)
		return h.createResource.Handle(ctx, CreateResourceCommand{
			ID:            newResourceID(),
			Name:          buildStackResourceName(namePrefix, id),
			Type:          compType,
			Image:         image,
			Tag:           tag,
			Config:        config,
			EnvironmentID: cmd.EnvironmentID,
			CreatedBy:     cmd.CreatedBy,
			NodeID:        cmd.NodeID,
			Replicas:      1,
			Ports:         comp.Ports,
			EnvVars:       envVars,
		})
	}

	// Pass 1: create & run job components synchronously (fail fast).
	var jobResources []*domain.Resource
	for _, comp := range cmd.CustomComponents {
		if comp.Type != domain.ResourceTypeJob {
			continue
		}
		resource, err := createComp(comp, domain.ResourceTypeJob)
		if err != nil {
			return nil, fmt.Errorf("create job component %s: %w", comp.ID, err)
		}
		if resource == nil {
			continue
		}
		if h.jobRunner != nil {
			if err := h.jobRunner.RunJobSync(ctx, resource.ID); err != nil {
				return nil, fmt.Errorf("job %s: %w", comp.ID, err)
			}
		}
		jobResources = append(jobResources, resource)
	}

	// Pass 2: create service components (not started yet — user triggers manually).
	var serviceResources []*domain.Resource
	for _, comp := range cmd.CustomComponents {
		if comp.Type == domain.ResourceTypeJob {
			continue
		}
		resource, err := createComp(comp, domain.ResourceTypeService)
		if err != nil {
			return nil, fmt.Errorf("create stack component %s: %w", comp.ID, err)
		}
		if resource == nil {
			continue
		}
		serviceResources = append(serviceResources, resource)
	}

	resources := append(jobResources, serviceResources...)
	if len(resources) == 0 {
		return nil, domain.NewUserFacingError("No stack components were enabled")
	}

	return &CreateResourceStackResult{
		TemplateID: template.ID,
		Resources:  resources,
	}, nil
}

func parsePortString(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	var port int
	_, err := fmt.Sscanf(value, "%d", &port)
	return port, err
}

func mergeEnvEntries(base []domain.ResourceStackTemplateEnvVar, overrides []ResourceEnvVarInput) []ResourceEnvVarInput {
	merged := make([]ResourceEnvVarInput, 0, len(base)+len(overrides))
	index := make(map[string]int, len(base)+len(overrides))

	for _, entry := range base {
		key := strings.TrimSpace(entry.Key)
		if key == "" {
			continue
		}
		index[key] = len(merged)
		merged = append(merged, ResourceEnvVarInput{
			Key:      key,
			Value:    entry.Value,
			IsSecret: entry.IsSecret,
		})
	}

	for _, entry := range overrides {
		key := strings.TrimSpace(entry.Key)
		if key == "" {
			continue
		}
		if idx, ok := index[key]; ok {
			merged[idx] = ResourceEnvVarInput{
				Key:      key,
				Value:    entry.Value,
				IsSecret: entry.IsSecret,
			}
			continue
		}
		index[key] = len(merged)
		merged = append(merged, ResourceEnvVarInput{
			Key:      key,
			Value:    entry.Value,
			IsSecret: entry.IsSecret,
		})
	}

	slices.SortFunc(merged, func(a, b ResourceEnvVarInput) int {
		return strings.Compare(a.Key, b.Key)
	})

	return merged
}

// mergeEnvVarInputs merges component-level env into the base (shared) env.
// Component env takes precedence over shared env for duplicate keys.
func mergeEnvVarInputs(base, overrides []ResourceEnvVarInput) []ResourceEnvVarInput {
	merged := make([]ResourceEnvVarInput, 0, len(base)+len(overrides))
	index := make(map[string]int, len(base)+len(overrides))
	for _, e := range base {
		k := strings.TrimSpace(e.Key)
		if k == "" {
			continue
		}
		index[k] = len(merged)
		merged = append(merged, e)
	}
	for _, e := range overrides {
		k := strings.TrimSpace(e.Key)
		if k == "" {
			continue
		}
		if i, ok := index[k]; ok {
			merged[i] = e
		} else {
			index[k] = len(merged)
			merged = append(merged, e)
		}
	}
	return merged
}

func buildStackResourceName(prefix, componentID string) string {
	prefix = strings.TrimSpace(prefix)
	componentID = strings.TrimSpace(componentID)
	if prefix == "" {
		return componentID
	}
	if componentID == "" {
		return prefix
	}
	return prefix + "-" + componentID
}
