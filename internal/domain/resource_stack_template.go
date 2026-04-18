package domain

type ResourceStackTemplatePort struct {
	Host      string `json:"host"`
	Container string `json:"container"`
}

type ResourceStackTemplateEnvVar struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret,omitempty"`
}

type ResourceStackTemplateComponent struct {
	ID             string                        `json:"id"`
	Name           string                        `json:"name"`
	Description    string                        `json:"description"`
	Type           string                        `json:"type"`
	Image          string                        `json:"image,omitempty"`
	Tag            string                        `json:"tag,omitempty"`
	Required       bool                          `json:"required"`
	DefaultEnabled bool                          `json:"default_enabled"`
	Ports          []ResourceStackTemplatePort   `json:"ports"`
	Env            []ResourceStackTemplateEnvVar `json:"env"`
	Volumes        []string                      `json:"volumes,omitempty"`
	Cmd            []string                      `json:"cmd,omitempty"`
	VolumeFiles    []VolumeFileTemplate          `json:"volume_files,omitempty"`
}

type ResourceStackTemplate struct {
	ID          string                           `json:"id"`
	Name        string                           `json:"name"`
	IconURL     string                           `json:"icon_url"`
	Image       string                           `json:"image"`
	Description string                           `json:"description"`
	Color       string                           `json:"color"`
	Abbr        string                           `json:"abbr"`
	Tags        []string                         `json:"tags"`
	SharedEnv   []ResourceStackTemplateEnvVar    `json:"shared_env"`
	Components  []ResourceStackTemplateComponent `json:"components"`
}
