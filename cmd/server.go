package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Abraxas-365/relay/pkg/errx"
	"github.com/Abraxas-365/relay/pkg/logx"
	"github.com/Abraxas-365/relay/recruitment/application/applicationapi"
	"github.com/Abraxas-365/relay/recruitment/candidate/candidateapi"
	"github.com/Abraxas-365/relay/recruitment/candidate/candidateauth"
	"github.com/Abraxas-365/relay/recruitment/job/jobapi"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// 1. Initialize Logger
	logx.SetLevel(logx.LevelInfo)
	logx.Info("Starting Relay API Server...")

	// 2. Initialize Dependency Container
	container := NewContainer()
	defer container.DB.Close()
	defer container.Redis.Close()

	// 3. Create Fiber App with Config
	app := fiber.New(fiber.Config{
		AppName:               "Relay ATS API",
		DisableStartupMessage: true,
		ErrorHandler:          globalErrorHandler,
	})

	// 4. Global Middleware
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // Configure for production
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-API-Key",
		AllowMethods: "GET, POST, PUT, DELETE, PATCH, HEAD",
	}))
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	// 5. Health Check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"db":     container.DB.Ping() == nil,
			"redis":  container.Redis.Ping(c.Context()).Err() == nil,
		})
	})

	// 6. Register Routes

	// --- Core Auth Routes ---
	// /auth/login, /auth/refresh, /auth/logout, /auth/me
	container.AuthService.RegisterRoutes(app)

	// --- IAM Routes ---
	// /api-keys/*
	container.APIKeyHandlers.RegisterRoutes(app, container.AuthMiddleware)

	// /invitations/*
	container.InvitationHandlers.RegisterRoutes(app, container.AuthMiddleware)

	// --- Recruitment Routes ---

	// Jobs: /api/jobs
	jobapi.RegisterRoutes(app, container.JobHandlers, container.UnifiedAuthMiddleware)

	// Candidates (Admin/Recruiter API): /api/candidates
	candidateapi.RegisterRoutes(app, container.CandidateHandlers, container.UnifiedAuthMiddleware)

	// Applications: /api/applications
	applicationapi.RegisterRoutes(app, container.ApplicationHandlers, container.UnifiedAuthMiddleware)

	// --- Candidate Portal Routes (Public/OTP) ---
	// /api/candidates/auth/*
	candidateTokenSvc := candidateauth.NewCandidateTokenService(container.TokenService)
	candidateAuthMiddleware := candidateauth.Middleware(candidateTokenSvc)
	candidateauth.RegisterRoutes(app, container.CandidateAuthHandlers, candidateAuthMiddleware)

	// 7. Start Server with Graceful Shutdown
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Run server in a goroutine
	go func() {
		logx.Infof("Server listening on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			logx.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c // Wait for signal
	logx.Info("Shutting down server...")

	if err := app.Shutdown(); err != nil {
		logx.Errorf("Server forced to shutdown: %v", err)
	}

	logx.Info("Server exited")
}

// globalErrorHandler converts internal errors to standard HTTP responses
func globalErrorHandler(c *fiber.Ctx, err error) error {
	// If it's a Fiber error (e.g., 404 handler not found)
	if e, ok := err.(*fiber.Error); ok {
		return c.Status(e.Code).JSON(fiber.Map{
			"error": e.Message,
			"code":  e.Code,
		})
	}

	// If it's our custom errx.Error
	if e, ok := err.(*errx.Error); ok {
		return c.Status(e.HTTPStatus).JSON(e.ToHTTPResponse())
	}

	// Default unknown error
	logx.Errorf("Internal Server Error: %v", err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error":   "Internal Server Error",
		"type":    "INTERNAL",
		"code":    "INTERNAL_ERROR",
		"message": "An unexpected error occurred",
	})
}

