package command

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"tango/internal/domain"
)

type CreateResourceStackCommand struct {
	TemplateID        string
	NamePrefix        string
	Image             string
	Tag               string
	EnvironmentID     string
	CreatedBy         string
	NodeID            *string
	EnabledComponents []string
	SharedEnvVars     []ResourceEnvVarInput
}

type CreateResourceStackResult struct {
	TemplateID string
	Resources  []*domain.Resource
}

type CreateResourceStackHandler struct {
	createResource *CreateResourceHandler
	templates      map[string]domain.ResourceStackTemplate
}

func NewCreateResourceStackHandler(
	createResource *CreateResourceHandler,
	templates []domain.ResourceStackTemplate,
) *CreateResourceStackHandler {
	index := make(map[string]domain.ResourceStackTemplate, len(templates))
	for _, template := range templates {
		index[strings.TrimSpace(template.ID)] = template
	}
	return &CreateResourceStackHandler{
		createResource: createResource,
		templates:      index,
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

	enabledSet := make(map[string]struct{}, len(cmd.EnabledComponents))
	for _, id := range cmd.EnabledComponents {
		key := strings.TrimSpace(id)
		if key == "" {
			continue
		}
		enabledSet[key] = struct{}{}
	}

	for id := range enabledSet {
		if !stackHasComponent(template, id) {
			return nil, domain.NewUserFacingError("Unknown stack component: " + id)
		}
	}

	sharedEnv := mergeEnvEntries(template.SharedEnv, cmd.SharedEnvVars)

	resources := make([]*domain.Resource, 0, len(template.Components))
	for _, component := range template.Components {
		if !component.Required {
			if len(enabledSet) == 0 {
				if !component.DefaultEnabled {
					continue
				}
			} else if _, ok := enabledSet[component.ID]; !ok {
				continue
			}
		}

		componentEnv := mergeEnvEntries(component.Env, nil)
		envVars := append(sharedEnv, componentEnv...)
		ports, err := stackPortsToInput(component.Ports)
		if err != nil {
			return nil, err
		}

		config := map[string]any{}
		if len(component.Volumes) > 0 {
			config["volumes"] = component.Volumes
		}
		if len(component.Cmd) > 0 {
			config["cmd"] = component.Cmd
		}
		if len(config) == 0 {
			config = nil
		}

		resource, err := h.createResource.Handle(ctx, CreateResourceCommand{
			ID:            newResourceID(),
			Name:          buildStackResourceName(namePrefix, component.ID),
			Type:          component.Type,
			Image:         image,
			Tag:           tag,
			Config:        config,
			EnvironmentID: cmd.EnvironmentID,
			CreatedBy:     cmd.CreatedBy,
			NodeID:        cmd.NodeID,
			Replicas:      1,
			Ports:         ports,
			EnvVars:       envVars,
		})
		if err != nil {
			return nil, fmt.Errorf("create stack component %s: %w", component.Name, err)
		}
		resources = append(resources, resource)
	}

	if len(resources) == 0 {
		return nil, domain.NewUserFacingError("No stack components were enabled")
	}

	return &CreateResourceStackResult{
		TemplateID: template.ID,
		Resources:  resources,
	}, nil
}

func stackHasComponent(template domain.ResourceStackTemplate, id string) bool {
	for _, component := range template.Components {
		if component.ID == id {
			return true
		}
	}
	return false
}

func stackPortsToInput(ports []domain.ResourceStackTemplatePort) ([]ResourcePortInput, error) {
	out := make([]ResourcePortInput, 0, len(ports))
	for _, port := range ports {
		hostPort, err := parsePortString(port.Host)
		if err != nil {
			return nil, domain.NewUserFacingError("Invalid host port for stack component")
		}
		containerPort, err := parsePortString(port.Container)
		if err != nil {
			return nil, domain.NewUserFacingError("Invalid container port for stack component")
		}
		out = append(out, ResourcePortInput{
			HostPort:     hostPort,
			InternalPort: containerPort,
			Proto:        "tcp",
		})
	}
	return out, nil
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
