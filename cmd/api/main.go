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

	"tango/internal/application/command"
	"tango/internal/application/query"
	"tango/internal/auth"
	"tango/internal/config"
	"tango/internal/domain"
	"tango/internal/handler/rest"
	response "tango/internal/handler/rest/response"
	infacache "tango/internal/infrastructure/cache"
	infradb "tango/internal/infrastructure/db"
	infradocker "tango/internal/infrastructure/docker"
	"tango/internal/infrastructure/persistence/models"
	persistrepo "tango/internal/infrastructure/persistence/repositories"
	infrahttp "tango/internal/infrastructure/server"
	infraservices "tango/internal/infrastructure/services"
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
	sourceProviderRepo := persistrepo.NewSourceProviderRepository(db)
	sourceConnectionRepo := persistrepo.NewSourceConnectionRepository(db)
	platformConfigRepo := persistrepo.NewPlatformConfigRepository(db)
	resourceDomainRepo := persistrepo.NewResourceDomainRepository(db)
	baseDomainRepo := persistrepo.NewBaseDomainRepository(db)

	bootstrapPlatformConfig(ctx, cfg, platformConfigRepo, logger)

	if err := auth.SeedDemoData(ctx, userRepo, roleRepo); err != nil {
		fatal(logger, "seed demo data failed", err)
	}

	if cfg.LLMConfigEncryptionKey == "" {
		fatal(logger, "LLM_CONFIG_ENCRYPTION_KEY is required", nil)
	}
	cipherService, err := infraservices.NewAESSecretCipher(cfg.LLMConfigEncryptionKey)
	if err != nil {
		fatal(logger, "init cipher failed", err)
	}

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
	if dr, err := infradocker.NewRepository(); err != nil {
		logger.Warn("docker unavailable, /docker endpoints disabled", "err", err)
	} else {
		dockerRepo = dr
		defer func() { _ = dr.Close() }()
		dockerWSHandler = rest.NewDockerWSHandler(dr)
		dockerHandler = rest.NewDockerHandler(
			query.NewListContainersHandler(dr),
			query.NewListImagesHandler(dr),
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

	// HTTP handlers
	authHandler := auth.NewHandler(userRepo)
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
	createResourceFromGitHandler := command.NewCreateResourceFromGitHandler(resourceRepo, buildJobRepo, buildSvc, resolveSourceConnectionTokenHandler)
	startBuildForResourceHandler := command.NewStartBuildForResourceHandler(resourceRepo, buildJobRepo, buildSvc, resolveSourceConnectionTokenHandler)
	updateResourceHandler := command.NewUpdateResourceHandler(resourceRepo, platformConfigRepo)

	// Traefik file provider — optional, only active when TRAEFIK_CONFIG_DIR is set
	var traefikFileProvider domain.TraefikFileProvider
	if cfg.TraefikConfigDir != "" {
		traefikFileProvider = infratraefik.NewFileProvider(cfg.TraefikConfigDir)
	}

	resourceRunSvc := infraservices.NewResourceRunService(resourceRepo, resourceRunRepo, dockerRepo, resourceDomainRepo, platformConfigRepo, traefikFileProvider, logger)
	buildSvc.SetResourceAutoStarter(resourceRunSvc)
	createStartResourceRunHandler := command.NewCreateStartResourceRunHandler(resourceRepo, resourceRunRepo, resourceRunSvc)
	stopResourceHandler := command.NewStopResourceHandler(resourceRepo, dockerRepo, traefikFileProvider)
	deleteResourceHandler := command.NewDeleteResourceHandler(resourceRepo, dockerRepo, traefikFileProvider)
	setEnvVarsHandler := command.NewSetResourceEnvVarsHandler(resourceRepo)
	listProjectsHandler := query.NewListProjectsHandler(projectRepo, environmentRepo)
	getProjectHandler := query.NewGetProjectHandler(projectRepo, environmentRepo, resourceRepo)
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
	settingsHandler := rest.NewSettingsHandler(platformConfigRepo, traefikFileProvider)
	baseDomainHandler := rest.NewBaseDomainHandler(baseDomainRepo, resourceDomainRepo, resourceRepo)

	// Write app Traefik config on every startup (picks up any env-seeded domain)
	if traefikFileProvider != nil {
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
		createResourceFromGitHandler,
		startBuildForResourceHandler,
		updateResourceHandler,
		createStartResourceRunHandler,
		stopResourceHandler,
		deleteResourceHandler,
		setEnvVarsHandler,
		listProjectsHandler,
		getProjectHandler,
		listEnvResourcesHandler,
		getResourceHandler,
		dockerRepo,
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
			sourceConnectionHandler.RegisterProtected(protected)
			settingsHandler.RegisterRoutes(protected)
			baseDomainHandler.RegisterRoutes(protected)
			if dockerHandler != nil {
				dockerHandler.Register(protected)
				dockerWSHandler.Register(protected)
			}
		}
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
	seedPlatformConfigIfMissing(ctx, repo, logger, domain.PlatformConfigAppDomain, cfg.AppDomain)
	appTLS := "false"
	if cfg.AppTLSEnabled {
		appTLS = "true"
	}
	seedPlatformConfigIfMissing(ctx, repo, logger, domain.PlatformConfigAppTLSEnabled, appTLS)
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
