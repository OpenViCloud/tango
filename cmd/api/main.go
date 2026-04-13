package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tango/internal/application/command"
	"tango/internal/application/query"
	appservices "tango/internal/application/services"
	"tango/internal/auth"
	"tango/internal/config"
	"tango/internal/domain"
	"tango/internal/handler/rest"
	response "tango/internal/handler/rest/response"
	infraansible "tango/internal/infrastructure/ansible"
	infacache "tango/internal/infrastructure/cache"
	infracf "tango/internal/infrastructure/cloudflare"
	infradb "tango/internal/infrastructure/db"
	infradocker "tango/internal/infrastructure/docker"
	infrakube "tango/internal/infrastructure/kube"
	"tango/internal/infrastructure/persistence/models"
	persistrepo "tango/internal/infrastructure/persistence/repositories"
	infrahttp "tango/internal/infrastructure/server"
	infraservices "tango/internal/infrastructure/services"
	infrassh "tango/internal/infrastructure/ssh"
	infratraefik "tango/internal/infrastructure/traefik"
	"tango/internal/messaging/inbound"

	docs "tango/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//go:embed all:static
var staticFiles embed.FS

func hasEmbeddedFrontend() bool {
	_, err := staticFiles.Open("static/index.html")
	return err == nil
}

// @title Tango API
// @version 0.1.0
// @description REST API for Tango authentication, users, and streaming endpoints.
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	logService, err := infraservices.NewLogService(infraservices.LogConfig{
		Level:      slog.LevelInfo,
		Format:     infraservices.LogFormat(cfg.LogFormat),
		Output:     infraservices.LogOutputMode(cfg.LogOutput),
		FilePath:   cfg.LogFilePath,
		MaxSizeMB:  cfg.LogMaxSizeMB,
		MaxBackups: cfg.LogMaxBackups,
		MaxAgeDays: cfg.LogMaxAgeDays,
		Compress:   cfg.LogCompress,
	})
	if err != nil {
		panic(err)
	}
	logger := logService.Logger()
	slog.SetDefault(logger)
	logger.Info("config loaded",
		"port", cfg.Port,
		"baseUrl", cfg.BaseURL,
		"frontendBaseUrl", cfg.FrontendBaseURL,
		"dbDriver", cfg.DBDriver,
		"backupRunnerBaseUrl", cfg.BackupRunnerBaseURL,
		"postgresInstallDir", cfg.PostgresInstallDir,
		"mysqlInstallDir", cfg.MySQLInstallDir,
		"mongoToolsDir", cfg.MongoToolsDir,
		"frontendEmbedded", hasEmbeddedFrontend(),
	)

	if err := infradb.EnsureDatabase(ctx, cfg.DBDriver, cfg.DBUrl); err != nil {
		fatal(logger, "ensure db failed", err)
	}
	db, err := infradb.Open(cfg.DBDriver, cfg.DBUrl)
	if err != nil {
		fatal(logger, "open db failed", err)
	}
	defer func() {
		if err := infradb.Close(db); err != nil {
			logger.Warn("close db failed", "err", err)
		}
	}()
	if err := infradb.Ping(ctx, db); err != nil {
		fatal(logger, "ping db failed", err)
	}
	if err := infradb.Migrate(ctx, db, models.All()...); err != nil {
		fatal(logger, "migrate db failed", err)
	}
	appCache, err := infacache.New(cfg)
	if err != nil {
		fatal(logger, "init cache failed", err)
	}
	_ = appCache

	userRepo := persistrepo.NewUserRepository(db)
	roleRepo := persistrepo.NewRoleRepository(db)
	channelRepo := persistrepo.NewChannelRepository(db)
	buildJobRepo := persistrepo.NewBuildJobRepository(db)
	projectRepo := persistrepo.NewProjectRepository(db)
	environmentRepo := persistrepo.NewEnvironmentRepository(db)
	resourceRepo := persistrepo.NewResourceRepository(db)
	resourceRunRepo := persistrepo.NewResourceRunRepository(db)
	databaseSourceRepo := persistrepo.NewDatabaseSourceRepository(db)
	storageRepo := persistrepo.NewStorageRepository(db)
	backupConfigRepo := persistrepo.NewBackupConfigRepository(db)
	backupRepo := persistrepo.NewBackupRepository(db)
	restoreRepo := persistrepo.NewRestoreRepository(db)
	sourceProviderRepo := persistrepo.NewSourceProviderRepository(db)
	sourceConnectionRepo := persistrepo.NewSourceConnectionRepository(db)
	platformConfigRepo := persistrepo.NewPlatformConfigRepository(db)
	resourceDomainRepo := persistrepo.NewResourceDomainRepository(db)
	baseDomainRepo := persistrepo.NewBaseDomainRepository(db)

	serverRepo := persistrepo.NewServerRepository(db)
	clusterRepo := persistrepo.NewClusterRepository(db)
	cloudflareConnectionRepo := persistrepo.NewCloudflareConnectionRepository(db)
	clusterTunnelRepo := persistrepo.NewClusterTunnelRepository(db)

	bootstrapPlatformConfig(ctx, cfg, platformConfigRepo, logger)

	if err := auth.SeedDemoData(ctx, userRepo, roleRepo); err != nil {
		fatal(logger, "seed demo data failed", err)
	}

	if cfg.DataEncryptionKey == "" {
		fatal(logger, "DATA_ENCRYPTION_KEY is required", nil)
	}
	cipherService, err := infraservices.NewAESSecretCipher(cfg.DataEncryptionKey)
	if err != nil {
		fatal(logger, "init cipher failed", err)
	}

	sshManager := infrassh.NewManager(platformConfigRepo, cipherService)
	if err := sshManager.EnsureKeypair(ctx); err != nil {
		logger.Warn("ensure SSH keypair failed", "err", err)
	}

	logBroadcaster := infraansible.NewLogBroadcaster()
	ansibleRunner := infraansible.NewRunner(logBroadcaster, sshManager)

	channelService := infraservices.NewChannelService(channelRepo, cipherService)
	roleService := infraservices.NewRoleService(roleRepo)
	integrationStateStore := infraservices.NewIntegrationStateStore(appCache)
	githubAppService := infraservices.NewGitHubAppService()

	buildSvc := infraservices.NewBuildService(infraservices.BuildConfig{
		BuildKitHost:     cfg.BuildKitHost,
		WorkspaceDir:     cfg.BuildWorkspaceDir,
		RegistryHost:     cfg.BuildRegistryHost,
		RegistryUsername: cfg.BuildRegistryUser,
		RegistryPassword: cfg.BuildRegistryPass,
	}, buildJobRepo, logger)

	publisher := inbound.NewService(logger)
	discordRuntime := infraservices.NewDiscordRuntimeService(ctx, publisher, logger)
	slackRuntime := infraservices.NewSlackRuntimeService(ctx, publisher, logger)
	whatsAppRuntime := infraservices.NewWhatsAppRuntimeService(ctx, publisher, logger)

	// Docker repository (optional — app starts fine if Docker is unavailable)
	var dockerHandler *rest.DockerHandler
	var dockerWSHandler *rest.DockerWSHandler
	var resourceTerminalWSHandler *rest.ResourceTerminalWSHandler
	var dockerRepo domain.DockerRepository
	var swarmRepo domain.SwarmRepository
	if dr, err := infradocker.NewRepository(); err != nil {
		logger.Warn("docker unavailable, /docker endpoints disabled", "err", err)
	} else {
		dockerRepo = dr
		swarmRepo = infradocker.NewSwarmRepository(dr)
		defer func() { _ = dr.Close() }()
		dockerWSHandler = rest.NewDockerWSHandler(dr)
		dockerHandler = rest.NewDockerHandler(
			query.NewListContainersHandler(dr),
			query.NewListImagesHandler(dr),
			query.NewGetContainerDetailsHandler(dr),
			query.NewGetContainerStatsHandler(dr),
			command.NewCreateContainerHandler(dr),
			command.NewStartContainerHandler(dr),
			command.NewStopContainerHandler(dr),
			command.NewRemoveContainerHandler(dr),
			command.NewPullImageHandler(dr),
			command.NewRemoveImageHandler(dr),
		)
	}

	// Command and query handlers
	statusHandler := query.NewGetStatusHandler()
	createUserHandler := command.NewCreateUserHandler(userRepo)
	updateUserHandler := command.NewUpdateUserHandler(userRepo)
	changePasswordHandler := command.NewChangePasswordHandler(userRepo)
	banUserHandler := command.NewBanUserHandler(userRepo)
	deleteUserHandler := command.NewDeleteUserHandler(userRepo)
	assignUserRoleHandler := command.NewAssignUserRoleHandler(userRepo, roleRepo)
	removeUserRoleHandler := command.NewRemoveUserRoleHandler(userRepo, roleRepo)
	getUserByIDHandler := query.NewGetUserByIDHandler(userRepo)
	listUsersHandler := query.NewListUsersHandler(userRepo)
	listUserRolesHandler := query.NewListUserRolesHandler(userRepo, roleRepo)
	createBuildJobHandler := command.NewCreateBuildJobHandler(buildJobRepo, buildSvc)
	createBuildJobFromUploadHandler := command.NewCreateBuildJobFromUploadHandler(buildJobRepo, buildSvc)
	cancelBuildJobHandler := command.NewCancelBuildJobHandler(buildJobRepo)
	getBuildJobHandler := query.NewGetBuildJobHandler(buildJobRepo)
	listBuildJobsHandler := query.NewListBuildJobsHandler(buildJobRepo)
	beginGitHubAppManifestHandler := command.NewBeginGitHubAppManifestHandler(integrationStateStore, githubAppService)
	completeGitHubAppManifestHandler := command.NewCompleteGitHubAppManifestHandler(sourceProviderRepo, integrationStateStore, githubAppService, cipherService)
	completeGitHubAppSetupHandler := command.NewCompleteGitHubAppSetupHandler(sourceProviderRepo, sourceConnectionRepo, integrationStateStore, githubAppService, cipherService)
	connectPATHandler := command.NewConnectPATHandler(sourceProviderRepo, sourceConnectionRepo, cipherService, githubAppService)
	resolveSourceConnectionTokenHandler := command.NewResolveSourceConnectionTokenHandler(sourceConnectionRepo, sourceProviderRepo, cipherService, githubAppService)
	listSourceConnectionsHandler := query.NewListSourceConnectionsHandler(sourceConnectionRepo)
	listGitHubRepositoriesHandler := query.NewListGitHubRepositoriesHandler(githubAppService)
	listGitHubUserRepositoriesHandler := query.NewListGitHubUserRepositoriesHandler(githubAppService)
	listGitHubBranchesHandler := query.NewListGitHubBranchesHandler(githubAppService)
	createDatabaseSourceHandler := command.NewCreateDatabaseSourceHandler(databaseSourceRepo, cipherService)
	updateDatabaseSourceHandler := command.NewUpdateDatabaseSourceHandler(databaseSourceRepo, cipherService)
	deleteDatabaseSourceHandler := command.NewDeleteDatabaseSourceHandler(databaseSourceRepo)
	listDatabaseSourcesHandler := query.NewListDatabaseSourcesHandler(databaseSourceRepo)
	getDatabaseSourceHandler := query.NewGetDatabaseSourceHandler(databaseSourceRepo)
	createStorageHandler := command.NewCreateStorageHandler(storageRepo, cipherService)
	updateStorageHandler := command.NewUpdateStorageHandler(storageRepo, cipherService)
	deleteStorageHandler := command.NewDeleteStorageHandler(storageRepo)
	listStoragesHandler := query.NewListStoragesHandler(storageRepo)
	getStorageHandler := query.NewGetStorageHandler(storageRepo)
	createBackupConfigHandler := command.NewCreateBackupConfigHandler(backupConfigRepo, databaseSourceRepo, storageRepo)
	updateBackupConfigHandler := command.NewUpdateBackupConfigHandler(backupConfigRepo, storageRepo)
	getBackupConfigHandler := query.NewGetBackupConfigHandler(backupConfigRepo)
	getBackupConfigBySourceHandler := query.NewGetBackupConfigByDatabaseSourceHandler(backupConfigRepo)
	backupRunnerClient := infraservices.NewBackupRunnerClient(cfg.BackupRunnerBaseURL, cfg.BackupRunnerToken)
	postgresBackupStrategy := infraservices.NewPostgresRunnerBackupStrategy(backupRunnerClient)
	postgresRestoreStrategy := infraservices.NewPostgresRunnerRestoreStrategy(backupRunnerClient)
	mysqlBackupStrategy := infraservices.NewMySQLRunnerBackupStrategy(backupRunnerClient)
	mysqlRestoreStrategy := infraservices.NewMySQLRunnerRestoreStrategy(backupRunnerClient)
	mariadbBackupStrategy := infraservices.NewMariaDBRunnerBackupStrategy(backupRunnerClient)
	mariadbRestoreStrategy := infraservices.NewMariaDBRunnerRestoreStrategy(backupRunnerClient)
	mongoBackupStrategy := infraservices.NewMongoRunnerBackupStrategy(backupRunnerClient)
	mongoRestoreStrategy := infraservices.NewMongoRunnerRestoreStrategy(backupRunnerClient)
	localBackupStorage := infraservices.NewLocalBackupStorage()
	backupStrategyResolver := infraservices.NewBackupStrategyResolver(mysqlBackupStrategy, mariadbBackupStrategy, postgresBackupStrategy, mongoBackupStrategy)
	restoreStrategyResolver := infraservices.NewRestoreStrategyResolver(mysqlRestoreStrategy, mariadbRestoreStrategy, postgresRestoreStrategy, mongoRestoreStrategy)
	storageDriverResolver := infraservices.NewStorageDriverResolver(localBackupStorage)
	backupExecutor := infraservices.NewBackupExecutor(backupRepo, databaseSourceRepo, backupConfigRepo, storageRepo, cipherService, backupStrategyResolver, storageDriverResolver)
	restoreExecutor := infraservices.NewRestoreExecutor(restoreRepo, backupRepo, databaseSourceRepo, storageRepo, cipherService, restoreStrategyResolver, storageDriverResolver)
	triggerBackupHandler := command.NewTriggerBackupHandler(databaseSourceRepo, backupConfigRepo, storageRepo, backupRepo, backupExecutor)
	listBackupsByDatabaseSourceHandler := query.NewListBackupsByDatabaseSourceHandler(backupRepo)
	getBackupHandler := query.NewGetBackupHandler(backupRepo)
	triggerRestoreHandler := command.NewTriggerRestoreHandler(backupRepo, restoreRepo, cipherService, restoreExecutor)
	getRestoreHandler := query.NewGetRestoreHandler(restoreRepo)

	// HTTP handlers
	authHandler := auth.NewHandler(userRepo, changePasswordHandler)
	userHandler := rest.NewUserHandler(
		createUserHandler,
		updateUserHandler,
		banUserHandler,
		deleteUserHandler,
		assignUserRoleHandler,
		removeUserRoleHandler,
		getUserByIDHandler,
		listUsersHandler,
		listUserRolesHandler,
	)
	discordRuntimeHandler := rest.NewDiscordRuntimeHandler(discordRuntime)
	roleHandler := rest.NewRoleHandler(roleService)
	buildHandler := rest.NewBuildHandler(createBuildJobHandler, createBuildJobFromUploadHandler, cancelBuildJobHandler, getBuildJobHandler, listBuildJobsHandler)
	buildWSHandler := rest.NewBuildWSHandler(buildSvc, getBuildJobHandler)
	logHandler := rest.NewLogHandler(logService)
	sourceConnectionHandler := rest.NewSourceConnectionHandler(
		beginGitHubAppManifestHandler,
		completeGitHubAppManifestHandler,
		completeGitHubAppSetupHandler,
		connectPATHandler,
		resolveSourceConnectionTokenHandler,
		listSourceConnectionsHandler,
		listGitHubRepositoriesHandler,
		listGitHubUserRepositoriesHandler,
		listGitHubBranchesHandler,
		platformConfigRepo,
		strings.TrimRight(cfg.FrontendBaseURL, "/")+"/projects",
		strings.TrimRight(cfg.BaseURL, "/"),
	)

	createProjectHandler := command.NewCreateProjectHandler(projectRepo)
	updateProjectHandler := command.NewUpdateProjectHandler(projectRepo)
	deleteProjectHandler := command.NewDeleteProjectHandler(projectRepo)
	createEnvironmentHandler := command.NewCreateEnvironmentHandler(environmentRepo)
	deleteEnvironmentHandler := command.NewDeleteEnvironmentHandler(environmentRepo)
	forkEnvironmentHandler := command.NewForkEnvironmentHandler(environmentRepo, resourceRepo)
	createResourceHandler := command.NewCreateResourceHandler(resourceRepo, dockerRepo, resourceDomainRepo, platformConfigRepo)
	resourceStackTemplates, err := query.LoadResourceStackTemplates()
	if err != nil {
		fatal(logger, "load resource stack templates failed", err)
	}
	// createResourceStackHandler is wired below after resourceRunSvc is available.
	createResourceFromGitHandler := command.NewCreateResourceFromGitHandler(resourceRepo, buildJobRepo, buildSvc, resolveSourceConnectionTokenHandler)
	startBuildForResourceHandler := command.NewStartBuildForResourceHandler(resourceRepo, buildJobRepo, buildSvc, resolveSourceConnectionTokenHandler)
	updateResourceHandler := command.NewUpdateResourceHandler(resourceRepo, platformConfigRepo)

	// Traefik file provider — optional, only active when TRAEFIK_CONFIG_DIR is set
	var traefikFileProvider domain.TraefikFileProvider
	var traefikRestarter domain.TraefikRestarter
	if cfg.TraefikConfigDir != "" {
		fp := infratraefik.NewFileProvider(cfg.TraefikConfigDir)
		traefikFileProvider = fp
		if r, err := infratraefik.NewRestarter("traefik"); err != nil {
			logger.Warn("traefik restarter unavailable", "err", err)
		} else {
			traefikRestarter = r
		}
	}

	resourceRunSvc := infraservices.NewResourceRunService(resourceRepo, resourceRunRepo, dockerRepo, swarmRepo, resourceDomainRepo, platformConfigRepo, traefikFileProvider, logger)
	createResourceStackHandler := command.NewCreateResourceStackHandler(createResourceHandler, resourceStackTemplates, resourceRunSvc)
	var runtimeReconciler appservices.ResourceRuntimeReconciler
	if dockerRepo != nil {
		runtimeReconciler = infraservices.NewResourceRuntimeReconciler(resourceRepo, dockerRepo, swarmRepo, logger)
	}
	buildSvc.SetResourceAutoStarter(resourceRunSvc)
	createStartResourceRunHandler := command.NewCreateStartResourceRunHandler(resourceRepo, resourceRunRepo, resourceRunSvc)
	stopResourceHandler := command.NewStopResourceHandler(resourceRepo, dockerRepo, swarmRepo, traefikFileProvider)
	deleteResourceHandler := command.NewDeleteResourceHandler(resourceRepo, dockerRepo, swarmRepo, traefikFileProvider)
	scaleResourceHandler := command.NewScaleResourceHandler(resourceRepo, swarmRepo)
	setEnvVarsHandler := command.NewSetResourceEnvVarsHandler(resourceRepo)
	listProjectsHandler := query.NewListProjectsHandler(projectRepo, environmentRepo)
	getProjectHandler := query.NewGetProjectHandler(projectRepo, environmentRepo, resourceRepo)
	listResourceTemplatesHandler, err := query.NewListResourceTemplatesHandler()
	if err != nil {
		fatal(logger, "load resource templates failed", err)
	}
	listResourceStackTemplatesHandler, err := query.NewListResourceStackTemplatesHandler()
	if err != nil {
		fatal(logger, "load resource stack templates failed", err)
	}
	listEnvResourcesHandler := query.NewListEnvironmentResourcesHandler(resourceRepo)
	getResourceHandler := query.NewGetResourceHandler(resourceRepo)
	telegramProjectNavigator := infraservices.NewTelegramProjectNavigator(
		listProjectsHandler,
		listEnvResourcesHandler,
		getResourceHandler,
		createStartResourceRunHandler,
		stopResourceHandler,
	)
	telegramRuntime := infraservices.NewTelegramRuntimeService(ctx, publisher, logger, telegramProjectNavigator)
	channelRuntimeService := infraservices.NewChannelRuntimeService(channelRepo, cipherService, discordRuntime, slackRuntime, telegramRuntime, whatsAppRuntime)
	defer func() {
		if err := discordRuntime.Stop(context.Background()); err != nil {
			logger.Warn("stop discord channel failed", "err", err)
		}
	}()
	defer func() {
		if err := slackRuntime.Stop(context.Background()); err != nil {
			logger.Warn("stop slack channel failed", "err", err)
		}
	}()
	defer func() {
		if err := telegramRuntime.Stop(context.Background()); err != nil {
			logger.Warn("stop telegram channel failed", "err", err)
		}
	}()
	defer func() {
		if err := whatsAppRuntime.Stop(context.Background()); err != nil {
			logger.Warn("stop whatsapp channel failed", "err", err)
		}
	}()
	if err := channelRuntimeService.StartActiveChannels(ctx); err != nil {
		fatal(logger, "bootstrap channels from db failed", err)
	}
	channelHandler := rest.NewChannelHandler(channelService, channelRuntimeService)
	getResourceRunHandler := query.NewGetResourceRunHandler(resourceRunRepo)
	resourceRunWSHandler := rest.NewResourceRunWSHandler(resourceRunSvc, getResourceRunHandler)
	if dockerRepo != nil {
		resourceTerminalWSHandler = rest.NewResourceTerminalWSHandler(dockerRepo, getResourceHandler)
	}
	serverHandler := rest.NewServerHandler(serverRepo, sshManager)
	clusterHandler := rest.NewClusterHandler(clusterRepo, serverRepo, ansibleRunner, cipherService)
	clusterWSHandler := rest.NewClusterWSHandler(logBroadcaster)
	kubeClientManager := infrakube.NewKubeClientManager(clusterRepo, cipherService)
	kubeHandler := rest.NewKubeHandler(kubeClientManager, clusterRepo)
	cfFactory := infracf.NewFactory()
	createCloudflareConnectionHandler := command.NewCreateCloudflareConnectionHandler(cloudflareConnectionRepo, cipherService, cfFactory)
	updateCloudflareConnectionHandler := command.NewUpdateCloudflareConnectionHandler(cloudflareConnectionRepo, cipherService, cfFactory)
	getCloudflareConnectionHandler := query.NewGetCloudflareConnectionHandler(cloudflareConnectionRepo)
	listCloudflareConnectionsHandler := query.NewListCloudflareConnectionsHandler(cloudflareConnectionRepo)
	cloudflareConnectionHandler := rest.NewCloudflareConnectionHandler(
		createCloudflareConnectionHandler,
		updateCloudflareConnectionHandler,
		getCloudflareConnectionHandler,
		listCloudflareConnectionsHandler,
	)
	exposeHandler := command.NewExposeServiceHandler(
		clusterRepo,
		clusterTunnelRepo,
		cloudflareConnectionRepo,
		kubeClientManager,
		cfFactory,
		cipherService,
	)
	unexposeHandler := command.NewUnexposeServiceHandler(
		clusterTunnelRepo,
		cloudflareConnectionRepo,
		kubeClientManager,
		cfFactory,
		cipherService,
	)
	tunnelHandler := rest.NewTunnelHandler(exposeHandler, unexposeHandler, clusterTunnelRepo, cloudflareConnectionRepo)

	settingsHandler := rest.NewSettingsHandler(platformConfigRepo, traefikFileProvider, traefikRestarter)
	baseDomainHandler := rest.NewBaseDomainHandler(baseDomainRepo, resourceDomainRepo, resourceRepo)
	backupHandler := rest.NewBackupHandler(
		createDatabaseSourceHandler,
		updateDatabaseSourceHandler,
		deleteDatabaseSourceHandler,
		listDatabaseSourcesHandler,
		getDatabaseSourceHandler,
		createStorageHandler,
		updateStorageHandler,
		deleteStorageHandler,
		listStoragesHandler,
		getStorageHandler,
		createBackupConfigHandler,
		updateBackupConfigHandler,
		getBackupConfigHandler,
		getBackupConfigBySourceHandler,
		triggerBackupHandler,
		listBackupsByDatabaseSourceHandler,
		getBackupHandler,
		triggerRestoreHandler,
		getRestoreHandler,
	)

	// Write Traefik static + dynamic config on every startup.
	if traefikFileProvider != nil {
		refreshTraefikStaticConfig(ctx, platformConfigRepo, traefikFileProvider, logger)
		refreshAppTraefikConfig(ctx, platformConfigRepo, traefikFileProvider, logger)
	}

	projectHandler := rest.NewProjectHandler(
		createProjectHandler,
		updateProjectHandler,
		deleteProjectHandler,
		createEnvironmentHandler,
		deleteEnvironmentHandler,
		forkEnvironmentHandler,
		createResourceHandler,
		createResourceStackHandler,
		createResourceFromGitHandler,
		startBuildForResourceHandler,
		updateResourceHandler,
		createStartResourceRunHandler,
		stopResourceHandler,
		deleteResourceHandler,
		scaleResourceHandler,
		setEnvVarsHandler,
		listProjectsHandler,
		getProjectHandler,
		listResourceTemplatesHandler,
		listResourceStackTemplatesHandler,
		listEnvResourcesHandler,
		getResourceHandler,
		runtimeReconciler,
		dockerRepo,
		swarmRepo,
		resourceDomainRepo,
		platformConfigRepo,
		traefikFileProvider,
		appCache,
	)

	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Title = "Tango API"
	docs.SwaggerInfo.Version = "0.1.0"
	docs.SwaggerInfo.Description = "REST API for Tango authentication, users, and streaming endpoints."

	r := gin.New()
	r.Use(response.Middleware(logger))
	r.Use(response.RequestLogger(logger))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ── API routes ──────────────────────────────
	api := r.Group("/api")
	{
		api.GET("/status", func(c *gin.Context) {
			c.JSON(200, statusHandler.Handle(c.Request.Context()))
		})

		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/refresh", authHandler.Refresh)
		api.POST("/auth/logout", authHandler.Logout)
		sourceConnectionHandler.RegisterPublic(api)

		protected := api.Group("/")
		protected.Use(auth.Middleware())
		{
			protected.POST("/auth/change-password", authHandler.ChangePassword)
			userHandler.Register(protected)
			channelHandler.Register(protected)
			discordRuntimeHandler.Register(protected)
			roleHandler.Register(protected)
			buildHandler.Register(protected)
			buildWSHandler.Register(protected)
			resourceRunWSHandler.Register(protected)
			if resourceTerminalWSHandler != nil {
				resourceTerminalWSHandler.Register(protected)
			}
			logHandler.Register(protected)
			projectHandler.Register(protected)
			backupHandler.Register(protected)
			sourceConnectionHandler.RegisterProtected(protected)
			serverHandler.Register(protected)
			clusterHandler.Register(protected)
			clusterWSHandler.Register(protected)
			kubeHandler.Register(protected)
			cloudflareConnectionHandler.Register(protected)
			tunnelHandler.Register(protected)
			settingsHandler.RegisterRoutes(protected)
			baseDomainHandler.RegisterRoutes(protected)
			rest.NewSwarmHandler(swarmRepo).RegisterRoutes(protected)
			if dockerHandler != nil {
				dockerHandler.Register(protected)
				dockerWSHandler.Register(protected)
			}
		}
	}

	if runtimeReconciler != nil {
		go func() {
			reconcileCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			summary, err := runtimeReconciler.ReconcileAll(reconcileCtx)
			if err != nil {
				logger.Warn("resource runtime reconcile on startup failed", "err", err)
				return
			}
			logger.Info("resource runtime reconcile on startup completed",
				"checked", summary.Checked,
				"updated", summary.Updated,
				"running", summary.Running,
				"stopped", summary.Stopped,
				"errored", summary.Errored,
				"missing_containers", summary.MissingContainers,
			)
		}()
	}

	// ── SPA handler ─────────────────────────────
	if hasEmbeddedFrontend() {
		subFS, err := fs.Sub(staticFiles, "static")
		if err != nil {
			logger.Warn("frontend disabled", "reason", "open embedded static fs failed", "err", err)
		} else {
			fileServer := http.FileServer(http.FS(subFS))
			r.NoRoute(func(c *gin.Context) {
				path := c.Request.URL.Path

				if strings.HasPrefix(path, "/api/") {
					c.JSON(404, gin.H{"error": "route not found"})
					return
				}

				filePath := "static" + path
				if _, err := staticFiles.Open(filePath); err != nil {
					c.Request.URL.Path = "/"
				}

				fileServer.ServeHTTP(c.Writer, c.Request)
			})
		}
	} else {
		logger.Info("frontend disabled", "reason", "embedded static/index.html not found; serving API only")
	}

	fmt.Printf("Server running on :%s\n", cfg.Port)
	if err := infrahttp.Run(ctx, ":"+cfg.Port, r, logger); err != nil {
		fatal(logger, "server error", err)
	}
}

func bootstrapPlatformConfig(ctx context.Context, cfg *config.Config, repo domain.PlatformConfigRepository, logger *slog.Logger) {
	ip := cfg.PublicIP
	if ip == "" {
		var detectErr error
		ip, detectErr = detectPublicIP()
		if detectErr != nil {
			logger.Warn("could not detect public IP, defaulting to 127.0.0.1", "err", detectErr)
			ip = "127.0.0.1"
		}
	}

	seedPlatformConfigIfMissing(ctx, repo, logger, domain.PlatformConfigPublicIP, ip)
	seedPlatformConfigIfMissing(ctx, repo, logger, domain.PlatformConfigBaseDomain, "localhost")
	seedPlatformConfigIfMissing(ctx, repo, logger, domain.PlatformConfigWildcardEnabled, "true")
	seedPlatformConfigIfMissing(ctx, repo, logger, domain.PlatformConfigTraefikNetwork, cfg.TraefikDockerNetwork)
	seedPlatformConfigIfMissing(ctx, repo, logger, domain.PlatformConfigCertResolver, "letsencrypt")

	// ENV vars take priority on every startup so re-install / update propagates changes to DB.
	syncOrSeedPlatformConfig(ctx, repo, logger, domain.PlatformConfigAppDomain, cfg.AppDomain)
	appTLS := "false"
	if cfg.AppTLSEnabled {
		appTLS = "true"
	}
	syncOrSeedPlatformConfig(ctx, repo, logger, domain.PlatformConfigAppTLSEnabled, appTLS)
	syncOrSeedPlatformConfig(ctx, repo, logger, domain.PlatformConfigACMEEmail, cfg.ACMEEmail)

	seedPlatformConfigIfMissing(ctx, repo, logger, domain.PlatformConfigAppBackendURL, cfg.AppBackendURL)
	seedPlatformConfigIfMissing(ctx, repo, logger, domain.PlatformConfigResourceMountRoot, cfg.ResourceMountRoot)
	logger.Info("platform config seeded", "public_ip", ip, "app_domain", cfg.AppDomain)
}

func seedPlatformConfigIfMissing(ctx context.Context, repo domain.PlatformConfigRepository, logger *slog.Logger, key, value string) {
	if _, err := repo.Get(ctx, key); err == nil {
		return
	}
	if err := repo.Set(ctx, key, value); err != nil {
		logger.Warn("seed platform config failed", "key", key, "err", err)
	}
}

// syncOrSeedPlatformConfig writes value to DB unconditionally when value is non-empty
// (env var override), otherwise falls back to seed-if-missing behaviour.
// Use this for settings that install scripts pass via env vars so re-installs propagate.
func syncOrSeedPlatformConfig(ctx context.Context, repo domain.PlatformConfigRepository, logger *slog.Logger, key, value string) {
	if value != "" {
		if err := repo.Set(ctx, key, value); err != nil {
			logger.Warn("sync platform config failed", "key", key, "err", err)
		}
		return
	}
	seedPlatformConfigIfMissing(ctx, repo, logger, key, value)
}

func refreshTraefikStaticConfig(ctx context.Context, repo domain.PlatformConfigRepository, fp domain.TraefikFileProvider, logger *slog.Logger) {
	acmeEmail := ""
	if cfg, err := repo.Get(ctx, domain.PlatformConfigACMEEmail); err == nil {
		acmeEmail = cfg.Value
	}
	if err := fp.WriteStaticConfig(acmeEmail); err != nil {
		logger.Warn("write traefik static config failed", "err", err)
	} else {
		logger.Info("traefik static config written", "acme_email_set", acmeEmail != "")
	}
}

func refreshAppTraefikConfig(ctx context.Context, repo domain.PlatformConfigRepository, fp domain.TraefikFileProvider, logger *slog.Logger) {
	appDomain := ""
	appTLS := false
	certResolver := ""
	backendURL := "http://app:8080"

	if cfg, err := repo.Get(ctx, domain.PlatformConfigAppDomain); err == nil {
		appDomain = cfg.Value
	}
	if cfg, err := repo.Get(ctx, domain.PlatformConfigAppTLSEnabled); err == nil {
		appTLS = cfg.Value == "true"
	}
	if cfg, err := repo.Get(ctx, domain.PlatformConfigCertResolver); err == nil {
		certResolver = cfg.Value
	}
	if cfg, err := repo.Get(ctx, domain.PlatformConfigAppBackendURL); err == nil && cfg.Value != "" {
		backendURL = cfg.Value
	}

	if appDomain == "" {
		return
	}
	if err := fp.WriteAppConfig(appDomain, appTLS, certResolver, backendURL); err != nil {
		logger.Warn("write app traefik config failed", "err", err)
	} else {
		logger.Info("app traefik config written", "domain", appDomain, "tls", appTLS)
	}
}

func detectPublicIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	buf := make([]byte, 64)
	n, err := resp.Body.Read(buf)
	if err != nil && n == 0 {
		return "", err
	}
	return strings.TrimSpace(string(buf[:n])), nil
}

func fatal(logger *slog.Logger, message string, err error) {
	if logger == nil {
		logger = slog.Default()
	}
	if err != nil {
		logger.Error(message, "err", err)
	} else {
		logger.Error(message)
	}
	os.Exit(1)
}
