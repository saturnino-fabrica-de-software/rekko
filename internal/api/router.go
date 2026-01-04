package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/jackc/pgx/v5/pgxpool"

	swagger "github.com/go-swagno/swagno-fiber/swagger"
	"github.com/saturnino-fabrica-de-software/rekko/internal/admin"
	"github.com/saturnino-fabrica-de-software/rekko/internal/api/docs"
	"github.com/saturnino-fabrica-de-software/rekko/internal/api/handler"
	adminHandler "github.com/saturnino-fabrica-de-software/rekko/internal/api/handler/admin"
	superHandler "github.com/saturnino-fabrica-de-software/rekko/internal/api/handler/super"
	"github.com/saturnino-fabrica-de-software/rekko/internal/api/middleware"
	"github.com/saturnino-fabrica-de-software/rekko/internal/cache"
	"github.com/saturnino-fabrica-de-software/rekko/internal/metrics"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
	"github.com/saturnino-fabrica-de-software/rekko/internal/ratelimit"
	"github.com/saturnino-fabrica-de-software/rekko/internal/repository"
	"github.com/saturnino-fabrica-de-software/rekko/internal/service"
	"github.com/saturnino-fabrica-de-software/rekko/internal/usage"
	"github.com/saturnino-fabrica-de-software/rekko/internal/webhook"
	"github.com/saturnino-fabrica-de-software/rekko/internal/ws"
)

type Dependencies struct {
	TenantRepo       *repository.TenantRepository
	APIKeyRepo       *repository.APIKeyRepository
	FaceRepo         *repository.FaceRepository
	VerificationRepo *repository.VerificationRepository
	FaceProvider     provider.FaceProvider
	LastUsedWorker   *middleware.LastUsedWorker
	DB               *pgxpool.Pool
}

type Router struct {
	app               *fiber.App
	logger            *slog.Logger
	deps              *Dependencies
	rateLimiter       *middleware.RateLimiter
	wsHub             *ws.Hub
	webhookWorker     *webhook.Worker
	cancelWorker      context.CancelFunc
	cancelHub         context.CancelFunc
	cancelUsageWorker context.CancelFunc
}

func NewRouter(logger *slog.Logger, deps *Dependencies) *Router {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler(logger),
		AppName:      "Rekko API",
	})

	return &Router{
		app:    app,
		logger: logger,
		deps:   deps,
	}
}

func (r *Router) Setup() {
	// Global middlewares
	r.app.Use(requestid.New())
	r.app.Use(middleware.Recover(r.logger))
	r.app.Use(middleware.Logger(r.logger))
	r.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Tenant-ID",
	}))

	// Swagger documentation (no auth required)
	sw := docs.NewSwagger()
	swagger.SwaggerHandler(r.app, sw.MustToJson())

	// Health check endpoints (no auth required)
	healthHandler := handler.NewHealthHandler()
	r.app.Get("/health", healthHandler.Health)
	r.app.Get("/ready", healthHandler.Ready)

	// API v1 group with authentication
	v1 := r.app.Group("/v1")

	// Only configure authenticated routes if dependencies were provided
	if r.deps != nil {
		// Initialize WebSocket Hub
		r.wsHub = ws.NewHub()
		hubCtx, hubCancel := context.WithCancel(context.Background())
		r.cancelHub = hubCancel
		go r.wsHub.Run(hubCtx)

		// Initialize Webhook Service and Worker
		webhookService := webhook.NewService(r.deps.DB)
		r.webhookWorker = webhook.NewWorker(r.deps.DB, webhookService, r.logger)

		ctx, cancel := context.WithCancel(context.Background())
		r.cancelWorker = cancel
		go r.webhookWorker.Run(ctx)

		// Auth middleware
		authDeps := middleware.AuthDependencies{
			TenantRepo:     r.deps.TenantRepo,
			APIKeyRepo:     r.deps.APIKeyRepo,
			Logger:         r.logger,
			LastUsedWorker: r.deps.LastUsedWorker,
		}
		v1.Use(middleware.Auth(authDeps))

		// Rate limiting (per tenant) - must come after auth to have tenant context
		r.rateLimiter = middleware.NewRateLimiter(middleware.DefaultRateLimiterConfig())
		v1.Use(r.rateLimiter.Handler())

		// Usage repository (needed for both FaceHandler and UsageService)
		usageRepo := usage.NewRepository(r.deps.DB)

		// Search audit repository
		searchAuditRepo := repository.NewSearchAuditRepository(r.deps.DB)

		// Rate limiter for search endpoint
		searchRateLimiter := ratelimit.NewRateLimiter(r.deps.DB, time.Minute)

		// Face service
		faceService := service.NewFaceService(
			r.deps.FaceRepo,
			r.deps.VerificationRepo,
			searchAuditRepo,
			r.deps.FaceProvider,
			searchRateLimiter,
		)

		// Face handler with usage tracking
		faceHandler := handler.NewFaceHandler(faceService, usageRepo, r.logger)

		// Face routes
		v1.Post("/faces", faceHandler.Register)
		v1.Post("/faces/verify", faceHandler.Verify)
		v1.Post("/faces/search", faceHandler.Search)
		v1.Post("/faces/liveness", faceHandler.CheckLiveness)
		v1.Delete("/faces/:external_id", faceHandler.Delete)

		// Usage service
		pgCache := cache.NewPGCache(r.deps.DB)
		cacheAdapter := usage.NewCacheAdapter(pgCache)
		usageService := usage.NewService(usageRepo, webhookService, cacheAdapter, r.logger)

		// Usage handler
		usageHandler := handler.NewUsageHandler(usageService, r.logger)

		// Usage routes
		v1.Get("/usage", usageHandler.GetUsage)

		// Start usage quota check worker (every 5 minutes)
		usageWorker := usage.NewWorker(usageService, usageRepo, r.logger, 5*time.Minute)
		usageWorkerCtx, usageWorkerCancel := context.WithCancel(context.Background())
		r.cancelUsageWorker = usageWorkerCancel
		go usageWorker.Run(usageWorkerCtx)

		// WebSocket endpoint
		v1.Get("/ws", ws.UpgradeMiddleware(), ws.Handler(r.wsHub))

		// Admin routes
		adminGroup := v1.Group("/admin")
		r.setupAdminRoutes(adminGroup, webhookService)

		// Super Admin routes (JWT auth)
		r.setupSuperAdminRoutes(v1)
	}
}

func (r *Router) setupAdminRoutes(adminGroup fiber.Router, webhookService *webhook.Service) {
	// Admin service dependencies
	metricsRepo := metrics.NewRepository(r.deps.DB)
	adminService := admin.NewService(metricsRepo, r.deps.DB, r.logger)

	// Admin handlers
	usageHandler := adminHandler.NewMetricsUsageHandler(adminService, r.logger)
	performanceHandler := adminHandler.NewMetricsPerformanceHandler(adminService, r.logger)
	qualityHandler := adminHandler.NewMetricsQualityHandler(adminService, r.logger)
	webhooksHandler := adminHandler.NewWebhooksHandler(webhookService, r.logger)

	// Metrics group
	metricsGroup := adminGroup.Group("/metrics")

	// Usage metrics
	metricsGroup.Get("/faces", usageHandler.GetFacesMetrics)
	metricsGroup.Get("/operations", usageHandler.GetOperationsMetrics)
	metricsGroup.Get("/requests", usageHandler.GetRequestsMetrics)

	// Performance metrics
	metricsGroup.Get("/latency", performanceHandler.GetLatencyMetrics)
	metricsGroup.Get("/throughput", performanceHandler.GetThroughputMetrics)
	metricsGroup.Get("/errors", performanceHandler.GetErrorMetrics)

	// Quality metrics
	metricsGroup.Get("/quality", qualityHandler.GetQualityMetrics)
	metricsGroup.Get("/confidence", qualityHandler.GetConfidenceMetrics)
	metricsGroup.Get("/matches", qualityHandler.GetMatchMetrics)

	// Webhooks routes
	adminGroup.Get("/webhooks", webhooksHandler.List)
	adminGroup.Post("/webhooks", webhooksHandler.Create)
	adminGroup.Delete("/webhooks/:id", webhooksHandler.Delete)
}

func (r *Router) setupSuperAdminRoutes(v1Group fiber.Router) {
	// Admin service dependencies
	metricsRepo := metrics.NewRepository(r.deps.DB)
	adminService := admin.NewService(metricsRepo, r.deps.DB, r.logger)

	// JWT service for super admin authentication
	jwtService := admin.NewJWTService(
		"your-secret-key", // TODO: move to config
		"rekko-api",
		24*time.Hour,
	)

	// Super admin group with JWT authentication
	superGroup := v1Group.Group("/super")
	superGroup.Use(middleware.AdminAuth(
		middleware.AdminLevelSuper,
		middleware.AdminAuthDependencies{
			JWTService: jwtService,
			Logger:     r.logger,
		},
	))

	// Create super admin handlers
	superTenantsHandler := superHandler.NewTenantsHandler(adminService, r.logger)
	superSystemHandler := superHandler.NewSystemHandler(adminService, r.logger)
	superProvidersHandler := superHandler.NewProvidersHandler(adminService, r.logger)

	// Tenants routes
	superGroup.Get("/tenants", superTenantsHandler.ListTenants)
	superGroup.Get("/tenants/:id/metrics", superTenantsHandler.GetTenantMetrics)
	superGroup.Post("/tenants/:id/quota", superTenantsHandler.UpdateTenantQuota)

	// System routes
	superGroup.Get("/system/health", superSystemHandler.GetSystemHealth)
	superGroup.Get("/system/metrics", superSystemHandler.GetSystemMetrics)

	// Providers routes
	superGroup.Get("/providers", superProvidersHandler.GetProvidersStatus)
}

func (r *Router) App() *fiber.App {
	return r.app
}

func (r *Router) Listen(addr string) error {
	return r.app.Listen(addr)
}

func (r *Router) Shutdown() error {
	// Stop WebSocket hub
	if r.cancelHub != nil {
		r.cancelHub()
	}

	// Stop webhook worker
	if r.cancelWorker != nil {
		r.cancelWorker()
	}

	// Stop usage quota check worker
	if r.cancelUsageWorker != nil {
		r.cancelUsageWorker()
	}

	// Stop rate limiter cleanup goroutine
	if r.rateLimiter != nil {
		r.rateLimiter.Stop()
	}

	return r.app.Shutdown()
}
