package api

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	swagger "github.com/go-swagno/swagno-fiber/swagger"
	"github.com/saturnino-fabrica-de-software/rekko/internal/api/docs"
	"github.com/saturnino-fabrica-de-software/rekko/internal/api/handler"
	"github.com/saturnino-fabrica-de-software/rekko/internal/api/middleware"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
	"github.com/saturnino-fabrica-de-software/rekko/internal/repository"
	"github.com/saturnino-fabrica-de-software/rekko/internal/service"
)

type Dependencies struct {
	TenantRepo       *repository.TenantRepository
	APIKeyRepo       *repository.APIKeyRepository
	FaceRepo         *repository.FaceRepository
	VerificationRepo *repository.VerificationRepository
	FaceProvider     provider.FaceProvider
	LastUsedWorker   *middleware.LastUsedWorker
}

type Router struct {
	app    *fiber.App
	logger *slog.Logger
	deps   *Dependencies
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
		// Auth middleware
		authDeps := middleware.AuthDependencies{
			TenantRepo:     r.deps.TenantRepo,
			APIKeyRepo:     r.deps.APIKeyRepo,
			Logger:         r.logger,
			LastUsedWorker: r.deps.LastUsedWorker,
		}
		v1.Use(middleware.Auth(authDeps))

		// Rate limiting (per tenant) - must come after auth to have tenant context
		rateLimiter := middleware.NewRateLimiter(middleware.DefaultRateLimiterConfig())
		v1.Use(rateLimiter.Handler())

		// Face service
		faceService := service.NewFaceService(
			r.deps.FaceRepo,
			r.deps.VerificationRepo,
			r.deps.FaceProvider,
		)

		// Face handler
		faceHandler := handler.NewFaceHandler(faceService)

		// Face routes
		v1.Post("/faces", faceHandler.Register)
		v1.Post("/faces/verify", faceHandler.Verify)
		v1.Delete("/faces/:external_id", faceHandler.Delete)
	}
}

func (r *Router) App() *fiber.App {
	return r.app
}

func (r *Router) Listen(addr string) error {
	return r.app.Listen(addr)
}

func (r *Router) Shutdown() error {
	return r.app.Shutdown()
}
