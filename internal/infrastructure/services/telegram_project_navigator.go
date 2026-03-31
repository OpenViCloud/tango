package services

import (
	"context"

	"tango/internal/application/command"
	"tango/internal/application/query"
	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type telegramProjectNavigator struct {
	listProjects     *query.ListProjectsHandler
	listEnvResources *query.ListEnvironmentResourcesHandler
	getResource      *query.GetResourceHandler
	startResourceRun *command.CreateStartResourceRunHandler
	stopResource     *command.StopResourceHandler
}

func NewTelegramProjectNavigator(
	listProjects *query.ListProjectsHandler,
	listEnvResources *query.ListEnvironmentResourcesHandler,
	getResource *query.GetResourceHandler,
	startResourceRun *command.CreateStartResourceRunHandler,
	stopResource *command.StopResourceHandler,
) appservices.TelegramProjectNavigator {
	return &telegramProjectNavigator{
		listProjects:     listProjects,
		listEnvResources: listEnvResources,
		getResource:      getResource,
		startResourceRun: startResourceRun,
		stopResource:     stopResource,
	}
}

func (n *telegramProjectNavigator) ListProjects(ctx context.Context) ([]appservices.TelegramProject, error) {
	if n == nil || n.listProjects == nil {
		return nil, nil
	}

	projects, err := n.listProjects.Handle(ctx, query.ListProjectsQuery{})
	if err != nil {
		return nil, err
	}

	out := make([]appservices.TelegramProject, 0, len(projects))
	for _, project := range projects {
		if project == nil {
			continue
		}

		envs := make([]appservices.TelegramProjectEnvironment, 0, len(project.Environments))
		for _, env := range project.Environments {
			envs = append(envs, appservices.TelegramProjectEnvironment{
				ID:   env.ID,
				Name: env.Name,
			})
		}

		out = append(out, appservices.TelegramProject{
			ID:           project.ID,
			Name:         project.Name,
			Environments: envs,
		})
	}

	return out, nil
}

func (n *telegramProjectNavigator) ListEnvironmentResources(ctx context.Context, environmentID string) ([]appservices.TelegramResource, error) {
	if n == nil || n.listEnvResources == nil {
		return nil, nil
	}

	resources, err := n.listEnvResources.Handle(ctx, query.ListEnvironmentResourcesQuery{EnvironmentID: environmentID})
	if err != nil {
		return nil, err
	}

	out := make([]appservices.TelegramResource, 0, len(resources))
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		out = append(out, toTelegramResource(resource))
	}
	return out, nil
}

func (n *telegramProjectNavigator) GetResource(ctx context.Context, resourceID string) (appservices.TelegramResource, error) {
	if n == nil || n.getResource == nil {
		return appservices.TelegramResource{}, domain.ErrResourceNotFound
	}

	resource, err := n.getResource.Handle(ctx, query.GetResourceQuery{ID: resourceID})
	if err != nil {
		return appservices.TelegramResource{}, err
	}
	return toTelegramResource(resource), nil
}

func (n *telegramProjectNavigator) StartResource(ctx context.Context, resourceID string) error {
	if n == nil || n.startResourceRun == nil {
		return domain.ErrResourceNotFound
	}
	_, err := n.startResourceRun.Handle(ctx, command.CreateStartResourceRunCommand{ResourceID: resourceID})
	return err
}

func (n *telegramProjectNavigator) StopResource(ctx context.Context, resourceID string) error {
	if n == nil || n.stopResource == nil {
		return domain.ErrResourceNotFound
	}
	return n.stopResource.Handle(ctx, command.StopResourceCommand{ID: resourceID})
}

func (n *telegramProjectNavigator) RestartResource(ctx context.Context, resourceID string) error {
	if err := n.StopResource(ctx, resourceID); err != nil {
		return err
	}
	return n.StartResource(ctx, resourceID)
}

func toTelegramResource(resource *domain.Resource) appservices.TelegramResource {
	ports := make([]appservices.TelegramResourcePort, 0, len(resource.Ports))
	for _, port := range resource.Ports {
		ports = append(ports, appservices.TelegramResourcePort{
			HostPort:     port.HostPort,
			InternalPort: port.InternalPort,
			Proto:        port.Proto,
			Label:        port.Label,
		})
	}

	return appservices.TelegramResource{
		ID:            resource.ID,
		Name:          resource.Name,
		Type:          resource.Type,
		Status:        resource.Status,
		Image:         resource.Image,
		Tag:           resource.Tag,
		EnvironmentID: resource.EnvironmentID,
		ContainerID:   resource.ContainerID,
		Ports:         ports,
	}
}
