package rest

// Supplemental Swagger annotations for routes that are registered in code but
// were missing from the generated docs.

type swaggerGenericRequest struct {
	Name string `json:"name"`
}

// @Summary POST /backup-configs
// @Tags backup-configs
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /backup-configs [post]
func swaggerPostBackupConfigs() {}

// @Summary GET /backup-configs/{id}
// @Tags backup-configs
// @Security BearerAuth
// @Param id path string true "id"
// @Router /backup-configs/{id} [get]
func swaggerGetBackupConfigsID() {}

// @Summary PUT /backup-configs/{id}
// @Tags backup-configs
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /backup-configs/{id} [put]
func swaggerPutBackupConfigsID() {}

// @Summary POST /backup-sources
// @Tags backup-sources
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /backup-sources [post]
func swaggerPostBackupSources() {}

// @Summary GET /backup-sources
// @Tags backup-sources
// @Security BearerAuth
// @Router /backup-sources [get]
func swaggerGetBackupSources() {}

// @Summary GET /backup-sources/{id}
// @Tags backup-sources
// @Security BearerAuth
// @Param id path string true "id"
// @Router /backup-sources/{id} [get]
func swaggerGetBackupSourcesID() {}

// @Summary PUT /backup-sources/{id}
// @Tags backup-sources
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /backup-sources/{id} [put]
func swaggerPutBackupSourcesID() {}

// @Summary DELETE /backup-sources/{id}
// @Tags backup-sources
// @Security BearerAuth
// @Param id path string true "id"
// @Router /backup-sources/{id} [delete]
func swaggerDeleteBackupSourcesID() {}

// @Summary GET /backup-sources/{id}/backup-config
// @Tags backup-sources
// @Security BearerAuth
// @Param id path string true "id"
// @Router /backup-sources/{id}/backup-config [get]
func swaggerGetBackupSourcesIDBackupConfig() {}

// @Summary GET /backup-sources/{id}/backups
// @Tags backup-sources
// @Security BearerAuth
// @Param id path string true "id"
// @Router /backup-sources/{id}/backups [get]
func swaggerGetBackupSourcesIDBackups() {}

// @Summary POST /backup-sources/{id}/backups
// @Tags backup-sources
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /backup-sources/{id}/backups [post]
func swaggerPostBackupSourcesIDBackups() {}

// @Summary GET /backups/{id}
// @Tags backups
// @Security BearerAuth
// @Param id path string true "id"
// @Router /backups/{id} [get]
func swaggerGetBackupsID() {}

// @Summary POST /backups/{id}/restore
// @Tags backups
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /backups/{id}/restore [post]
func swaggerPostBackupsIDRestore() {}

// @Summary GET /restores/{id}
// @Tags restores
// @Security BearerAuth
// @Param id path string true "id"
// @Router /restores/{id} [get]
func swaggerGetRestoresID() {}

// @Summary GET /cloudflare/connections
// @Tags cloudflare
// @Security BearerAuth
// @Router /cloudflare/connections [get]
func swaggerGetCloudflareConnections() {}

// @Summary POST /cloudflare/connections
// @Tags cloudflare
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /cloudflare/connections [post]
func swaggerPostCloudflareConnections() {}

// @Summary GET /cloudflare/connections/{id}
// @Tags cloudflare
// @Security BearerAuth
// @Param id path string true "id"
// @Router /cloudflare/connections/{id} [get]
func swaggerGetCloudflareConnectionsID() {}

// @Summary PUT /cloudflare/connections/{id}
// @Tags cloudflare
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /cloudflare/connections/{id} [put]
func swaggerPutCloudflareConnectionsID() {}

// @Summary GET /clusters
// @Tags clusters
// @Security BearerAuth
// @Router /clusters [get]
func swaggerGetClusters() {}

// @Summary POST /clusters
// @Tags clusters
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /clusters [post]
func swaggerPostClusters() {}

// @Summary GET /clusters/{id}
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Router /clusters/{id} [get]
func swaggerGetClustersID() {}

// @Summary DELETE /clusters/{id}
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Router /clusters/{id} [delete]
func swaggerDeleteClustersID() {}

// @Summary POST /clusters/{id}/inventory-preview
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /clusters/{id}/inventory-preview [post]
func swaggerPostClustersIDInventoryPreview() {}

// @Summary GET /clusters/{id}/kubeconfig
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Router /clusters/{id}/kubeconfig [get]
func swaggerGetClustersIDKubeconfig() {}

// @Summary GET /clusters/{id}/namespaces
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Router /clusters/{id}/namespaces [get]
func swaggerGetClustersIDNamespaces() {}

// @Summary GET /clusters/{id}/pods
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Router /clusters/{id}/pods [get]
func swaggerGetClustersIDPods() {}

// @Summary POST /clusters/{id}/pods
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /clusters/{id}/pods [post]
func swaggerPostClustersIDPods() {}

// @Summary DELETE /clusters/{id}/pods/{name}
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Param name path string true "name"
// @Router /clusters/{id}/pods/{name} [delete]
func swaggerDeleteClustersIDPodsName() {}

// @Summary GET /clusters/{id}/services
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Router /clusters/{id}/services [get]
func swaggerGetClustersIDServices() {}

// @Summary POST /clusters/{id}/services
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /clusters/{id}/services [post]
func swaggerPostClustersIDServices() {}

// @Summary DELETE /clusters/{id}/services/{name}
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Param name path string true "name"
// @Router /clusters/{id}/services/{name} [delete]
func swaggerDeleteClustersIDServicesName() {}

// @Summary GET /clusters/{id}/volumes
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Router /clusters/{id}/volumes [get]
func swaggerGetClustersIDVolumes() {}

// @Summary GET /clusters/{id}/volume-claims
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Router /clusters/{id}/volume-claims [get]
func swaggerGetClustersIDVolumeClaims() {}

// @Summary GET /clusters/{id}/tunnels
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Router /clusters/{id}/tunnels [get]
func swaggerGetClustersIDTunnels() {}

// @Summary POST /clusters/{id}/tunnels/import
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /clusters/{id}/tunnels/import [post]
func swaggerPostClustersIDTunnelsImport() {}

// @Summary POST /clusters/{id}/tunnels/expose
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /clusters/{id}/tunnels/expose [post]
func swaggerPostClustersIDTunnelsExpose() {}

// @Summary DELETE /clusters/{id}/tunnels/expose
// @Tags clusters
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /clusters/{id}/tunnels/expose [delete]
func swaggerDeleteClustersIDTunnelsExpose() {}

// @Summary GET /docker/containers/{id}/stats
// @Tags docker
// @Security BearerAuth
// @Param id path string true "id"
// @Router /docker/containers/{id}/stats [get]
func swaggerGetDockerContainersIDStats() {}

// @Summary GET /docker/containers/{id}
// @Tags docker
// @Security BearerAuth
// @Param id path string true "id"
// @Router /docker/containers/{id} [get]
func swaggerGetDockerContainersID() {}

// @Summary GET /domains/check
// @Tags domains
// @Security BearerAuth
// @Param domain query string true "domain"
// @Router /domains/check [get]
func swaggerGetDomainsCheck() {}

// @Summary GET /projects
// @Tags projects
// @Security BearerAuth
// @Router /projects [get]
func swaggerGetProjects() {}

// @Summary POST /projects
// @Tags projects
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /projects [post]
func swaggerPostProjects() {}

// @Summary GET /projects/{id}
// @Tags projects
// @Security BearerAuth
// @Param id path string true "id"
// @Router /projects/{id} [get]
func swaggerGetProjectsID() {}

// @Summary PUT /projects/{id}
// @Tags projects
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /projects/{id} [put]
func swaggerPutProjectsID() {}

// @Summary DELETE /projects/{id}
// @Tags projects
// @Security BearerAuth
// @Param id path string true "id"
// @Router /projects/{id} [delete]
func swaggerDeleteProjectsID() {}

// @Summary POST /projects/{id}/environments
// @Tags projects
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /projects/{id}/environments [post]
func swaggerPostProjectsIDEnvironments() {}

// @Summary GET /resource-templates
// @Tags resources
// @Security BearerAuth
// @Router /resource-templates [get]
func swaggerGetResourceTemplates() {}

// @Summary GET /resource-stack-templates
// @Tags resources
// @Security BearerAuth
// @Router /resource-stack-templates [get]
func swaggerGetResourceStackTemplates() {}

// @Summary DELETE /environments/{envId}
// @Tags environments
// @Security BearerAuth
// @Param envId path string true "envId"
// @Router /environments/{envId} [delete]
func swaggerDeleteEnvironmentsEnvID() {}

// @Summary POST /environments/{envId}/fork
// @Tags environments
// @Security BearerAuth
// @Param envId path string true "envId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /environments/{envId}/fork [post]
func swaggerPostEnvironmentsEnvIDFork() {}

// @Summary GET /environments/{envId}/resources
// @Tags environments
// @Security BearerAuth
// @Param envId path string true "envId"
// @Router /environments/{envId}/resources [get]
func swaggerGetEnvironmentsEnvIDResources() {}

// @Summary POST /environments/{envId}/resources
// @Tags environments
// @Security BearerAuth
// @Param envId path string true "envId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /environments/{envId}/resources [post]
func swaggerPostEnvironmentsEnvIDResources() {}

// @Summary POST /environments/{envId}/resource-stacks
// @Tags environments
// @Security BearerAuth
// @Param envId path string true "envId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /environments/{envId}/resource-stacks [post]
func swaggerPostEnvironmentsEnvIDResourceStacks() {}

// @Summary POST /environments/{envId}/resources/from-git
// @Tags environments
// @Security BearerAuth
// @Param envId path string true "envId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /environments/{envId}/resources/from-git [post]
func swaggerPostEnvironmentsEnvIDResourcesFromGit() {}

// @Summary POST /resources/reconcile
// @Tags resources
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/reconcile [post]
func swaggerPostResourcesReconcile() {}

// @Summary GET /resources/{resourceId}
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Router /resources/{resourceId} [get]
func swaggerGetResourcesResourceID() {}

// @Summary PUT /resources/{resourceId}
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/{resourceId} [put]
func swaggerPutResourcesResourceID() {}

// @Summary DELETE /resources/{resourceId}
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Router /resources/{resourceId} [delete]
func swaggerDeleteResourcesResourceID() {}

// @Summary POST /resources/{resourceId}/build
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/{resourceId}/build [post]
func swaggerPostResourcesResourceIDBuild() {}

// @Summary POST /resources/{resourceId}/reconcile
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/{resourceId}/reconcile [post]
func swaggerPostResourcesResourceIDReconcile() {}

// @Summary GET /resources/{resourceId}/connection-info
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Router /resources/{resourceId}/connection-info [get]
func swaggerGetResourcesResourceIDConnectionInfo() {}

// @Summary POST /resources/{resourceId}/start
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/{resourceId}/start [post]
func swaggerPostResourcesResourceIDStart() {}

// @Summary POST /resources/{resourceId}/stop
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/{resourceId}/stop [post]
func swaggerPostResourcesResourceIDStop() {}

// @Summary POST /resources/{resourceId}/restart
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/{resourceId}/restart [post]
func swaggerPostResourcesResourceIDRestart() {}

// @Summary POST /resources/{resourceId}/scale
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/{resourceId}/scale [post]
func swaggerPostResourcesResourceIDScale() {}

// @Summary GET /resources/{resourceId}/logs
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Router /resources/{resourceId}/logs [get]
func swaggerGetResourcesResourceIDLogs() {}

// @Summary GET /resources/{resourceId}/env-vars
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Router /resources/{resourceId}/env-vars [get]
func swaggerGetResourcesResourceIDEnvVars() {}

// @Summary PUT /resources/{resourceId}/env-vars
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/{resourceId}/env-vars [put]
func swaggerPutResourcesResourceIDEnvVars() {}

// @Summary GET /resources/{resourceId}/domains
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Router /resources/{resourceId}/domains [get]
func swaggerGetResourcesResourceIDDomains() {}

// @Summary POST /resources/{resourceId}/domains
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/{resourceId}/domains [post]
func swaggerPostResourcesResourceIDDomains() {}

// @Summary PATCH /resources/{resourceId}/domains/{domainId}
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param domainId path string true "domainId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/{resourceId}/domains/{domainId} [patch]
func swaggerPatchResourcesResourceIDDomainsDomainID() {}

// @Summary DELETE /resources/{resourceId}/domains/{domainId}
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param domainId path string true "domainId"
// @Router /resources/{resourceId}/domains/{domainId} [delete]
func swaggerDeleteResourcesResourceIDDomainsDomainID() {}

// @Summary POST /resources/{resourceId}/domains/{domainId}/verify
// @Tags resources
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Param domainId path string true "domainId"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /resources/{resourceId}/domains/{domainId}/verify [post]
func swaggerPostResourcesResourceIDDomainsDomainIDVerify() {}

// @Summary GET /servers
// @Tags servers
// @Security BearerAuth
// @Router /servers [get]
func swaggerGetServers() {}

// @Summary POST /servers
// @Tags servers
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /servers [post]
func swaggerPostServers() {}

// @Summary GET /servers/ssh-public-key
// @Tags servers
// @Security BearerAuth
// @Router /servers/ssh-public-key [get]
func swaggerGetServersSSHPublicKey() {}

// @Summary GET /servers/{id}
// @Tags servers
// @Security BearerAuth
// @Param id path string true "id"
// @Router /servers/{id} [get]
func swaggerGetServersID() {}

// @Summary DELETE /servers/{id}
// @Tags servers
// @Security BearerAuth
// @Param id path string true "id"
// @Router /servers/{id} [delete]
func swaggerDeleteServersID() {}

// @Summary POST /servers/{id}/ping
// @Tags servers
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /servers/{id}/ping [post]
func swaggerPostServersIDPing() {}

// @Summary GET /settings
// @Tags settings
// @Security BearerAuth
// @Router /settings [get]
func swaggerGetSettings() {}

// @Summary PATCH /settings
// @Tags settings
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /settings [patch]
func swaggerPatchSettings() {}

// @Summary POST /settings/traefik/restart
// @Tags settings
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /settings/traefik/restart [post]
func swaggerPostSettingsTraefikRestart() {}

// @Summary GET /settings/base-domains
// @Tags settings
// @Security BearerAuth
// @Router /settings/base-domains [get]
func swaggerGetSettingsBaseDomains() {}

// @Summary POST /settings/base-domains
// @Tags settings
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /settings/base-domains [post]
func swaggerPostSettingsBaseDomains() {}

// @Summary DELETE /settings/base-domains/{id}
// @Tags settings
// @Security BearerAuth
// @Param id path string true "id"
// @Router /settings/base-domains/{id} [delete]
func swaggerDeleteSettingsBaseDomainsID() {}

// @Summary GET /source-connections
// @Tags source-connections
// @Security BearerAuth
// @Router /source-connections [get]
func swaggerGetSourceConnections() {}

// @Summary POST /source-connections/github/apps
// @Tags source-connections
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /source-connections/github/apps [post]
func swaggerPostSourceConnectionsGitHubApps() {}

// @Summary GET /source-connections/github/callback
// @Tags source-connections
// @Router /source-connections/github/callback [get]
func swaggerGetSourceConnectionsGitHubCallback() {}

// @Summary GET /source-connections/github/setup
// @Tags source-connections
// @Router /source-connections/github/setup [get]
func swaggerGetSourceConnectionsGitHubSetup() {}

// @Summary POST /source-connections/github/webhook
// @Tags source-connections
// @Param request body swaggerGenericRequest true "Payload"
// @Router /source-connections/github/webhook [post]
func swaggerPostSourceConnectionsGitHubWebhook() {}

// @Summary POST /source-connections/pat
// @Tags source-connections
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /source-connections/pat [post]
func swaggerPostSourceConnectionsPAT() {}

// @Summary GET /source-connections/{id}/repos
// @Tags source-connections
// @Security BearerAuth
// @Param id path string true "id"
// @Router /source-connections/{id}/repos [get]
func swaggerGetSourceConnectionsIDRepos() {}

// @Summary GET /source-connections/{id}/repos/{owner}/{repo}/branches
// @Tags source-connections
// @Security BearerAuth
// @Param id path string true "id"
// @Param owner path string true "owner"
// @Param repo path string true "repo"
// @Router /source-connections/{id}/repos/{owner}/{repo}/branches [get]
func swaggerGetSourceConnectionsIDReposOwnerRepoBranches() {}

// @Summary GET /status
// @Tags status
// @Router /status [get]
func swaggerGetStatus() {}

// @Summary GET /storages
// @Tags storages
// @Security BearerAuth
// @Router /storages [get]
func swaggerGetStorages() {}

// @Summary POST /storages
// @Tags storages
// @Security BearerAuth
// @Param request body swaggerGenericRequest true "Payload"
// @Router /storages [post]
func swaggerPostStorages() {}

// @Summary GET /storages/{id}
// @Tags storages
// @Security BearerAuth
// @Param id path string true "id"
// @Router /storages/{id} [get]
func swaggerGetStoragesID() {}

// @Summary PUT /storages/{id}
// @Tags storages
// @Security BearerAuth
// @Param id path string true "id"
// @Param request body swaggerGenericRequest true "Payload"
// @Router /storages/{id} [put]
func swaggerPutStoragesID() {}

// @Summary DELETE /storages/{id}
// @Tags storages
// @Security BearerAuth
// @Param id path string true "id"
// @Router /storages/{id} [delete]
func swaggerDeleteStoragesID() {}

// @Summary GET /swarm/status
// @Tags swarm
// @Security BearerAuth
// @Router /swarm/status [get]
func swaggerGetSwarmStatus() {}

// @Summary GET /swarm/nodes
// @Tags swarm
// @Security BearerAuth
// @Router /swarm/nodes [get]
func swaggerGetSwarmNodes() {}

// @Summary GET /ws/builds/{id}/logs
// @Tags websocket
// @Security BearerAuth
// @Param id path string true "id"
// @Router /ws/builds/{id}/logs [get]
func swaggerGetWSBuildsIDLogs() {}

// @Summary GET /ws/clusters/{id}/logs
// @Tags websocket
// @Security BearerAuth
// @Param id path string true "id"
// @Router /ws/clusters/{id}/logs [get]
func swaggerGetWSClustersIDLogs() {}

// @Summary GET /ws/docker/images/pull
// @Tags websocket
// @Security BearerAuth
// @Router /ws/docker/images/pull [get]
func swaggerGetWSDockerImagesPull() {}

// @Summary GET /ws/resource-runs/{id}/logs
// @Tags websocket
// @Security BearerAuth
// @Param id path string true "id"
// @Router /ws/resource-runs/{id}/logs [get]
func swaggerGetWSResourceRunsIDLogs() {}

// @Summary GET /ws/resources/{resourceId}/terminal
// @Tags websocket
// @Security BearerAuth
// @Param resourceId path string true "resourceId"
// @Router /ws/resources/{resourceId}/terminal [get]
func swaggerGetWSResourcesResourceIDTerminal() {}
