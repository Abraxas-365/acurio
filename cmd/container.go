package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Abraxas-365/relay/pkg/fsx"
	"github.com/Abraxas-365/relay/pkg/fsx/fsxs3"
	"github.com/Abraxas-365/relay/pkg/iam"
	"github.com/Abraxas-365/relay/pkg/iam/apikey/apikeyapi"
	"github.com/Abraxas-365/relay/pkg/iam/apikey/apikeyinfra"
	"github.com/Abraxas-365/relay/pkg/iam/apikey/apikeysrv"
	"github.com/Abraxas-365/relay/pkg/iam/auth"
	"github.com/Abraxas-365/relay/pkg/iam/auth/authinfra"
	"github.com/Abraxas-365/relay/pkg/iam/invitation/invitationapi"
	"github.com/Abraxas-365/relay/pkg/iam/invitation/invitationinfra"
	"github.com/Abraxas-365/relay/pkg/iam/invitation/invitationsrv"
	"github.com/Abraxas-365/relay/pkg/iam/otp/otpinfra"
	"github.com/Abraxas-365/relay/pkg/iam/otp/otpsrv"
	"github.com/Abraxas-365/relay/pkg/iam/role/roleinfra"
	"github.com/Abraxas-365/relay/pkg/iam/role/rolesrv"
	"github.com/Abraxas-365/relay/pkg/iam/tenant/tenantinfra"
	"github.com/Abraxas-365/relay/pkg/iam/tenant/tenantsrv"
	"github.com/Abraxas-365/relay/pkg/iam/user/userinfra"
	"github.com/Abraxas-365/relay/pkg/iam/user/usersrv"
	"github.com/Abraxas-365/relay/pkg/logx"
	"github.com/Abraxas-365/relay/recruitment/application/applicationapi"
	"github.com/Abraxas-365/relay/recruitment/application/applicationinfra"
	"github.com/Abraxas-365/relay/recruitment/application/applicationsrv"
	"github.com/Abraxas-365/relay/recruitment/candidate/candidateapi"
	"github.com/Abraxas-365/relay/recruitment/candidate/candidateauth"
	"github.com/Abraxas-365/relay/recruitment/candidate/candidateinfra"
	"github.com/Abraxas-365/relay/recruitment/candidate/candidatesrv"
	"github.com/Abraxas-365/relay/recruitment/job/jobapi"
	"github.com/Abraxas-365/relay/recruitment/job/jobinfra"
	"github.com/Abraxas-365/relay/recruitment/job/jobsrv"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Container holds all application dependencies
type Container struct {
	// Config
	AuthConfig auth.Config

	// Infrastructure
	DB         *sqlx.DB
	Redis      *redis.Client
	FileSystem fsx.FileSystem
	S3Client   *s3.Client

	// Core IAM Services
	AuthService       *auth.AuthHandlers
	TokenService      auth.TokenService
	APIKeyService     *apikeysrv.APIKeyService
	TenantService     *tenantsrv.TenantService
	UserService       *usersrv.UserService
	RoleService       *rolesrv.RoleService
	InvitationService *invitationsrv.InvitationService
	OTPService        *otpsrv.OTPService

	// Recruitment Services
	JobService           *jobsrv.JobService
	CandidateService     *candidatesrv.CandidateService
	CandidateAuthService *candidateauth.CandidateAuthService
	ApplicationService   *applicationsrv.ApplicationService

	// API Handlers
	APIKeyHandlers        *apikeyapi.APIKeyHandlers
	InvitationHandlers    *invitationapi.InvitationHandlers
	JobHandlers           *jobapi.Handlers
	CandidateHandlers     *candidateapi.Handlers
	CandidateAuthHandlers *candidateauth.Handlers
	ApplicationHandlers   *applicationapi.Handlers

	// Middleware
	UnifiedAuthMiddleware *auth.UnifiedAuthMiddleware
	AuthMiddleware        *auth.TokenMiddleware
}

// NewContainer initializes the dependency injection container
func NewContainer() *Container {
	c := &Container{}
	c.initInfrastructure()
	c.initRepositories()
	return c
}

func (c *Container) initInfrastructure() {
	// 1. Database Connection
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		logx.Fatalf("Failed to connect to database: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	c.DB = db

	// 2. Redis Connection
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPass := os.Getenv("REDIS_PASS")
	c.Redis = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       0,
	})
	if _, err := c.Redis.Ping(context.Background()).Result(); err != nil {
		logx.Warnf("Failed to connect to Redis: %v", err)
	}

	// 3. AWS S3 Configuration
	awsRegion := os.Getenv("AWS_REGION")
	awsBucket := os.Getenv("AWS_BUCKET")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if err != nil {
		logx.Fatalf("unable to load SDK config, %v", err)
	}
	c.S3Client = s3.NewFromConfig(cfg)
	c.FileSystem = fsxs3.NewS3FileSystem(c.S3Client, awsBucket, "uploads")

	// 4. Auth Config
	c.AuthConfig = auth.DefaultConfig()
	c.AuthConfig.JWT.SecretKey = os.Getenv("JWT_SECRET")
	if c.AuthConfig.JWT.SecretKey == "" {
		logx.Warn("JWT_SECRET is not set, using default (unsafe for production)")
		c.AuthConfig.JWT.SecretKey = "super-secret-key-please-change-me-in-production"
	}

	// OAuth Configs
	c.AuthConfig.OAuth.Google.ClientID = os.Getenv("GOOGLE_CLIENT_ID")
	c.AuthConfig.OAuth.Google.ClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	c.AuthConfig.OAuth.Google.RedirectURL = os.Getenv("GOOGLE_REDIRECT_URL")
}

func (c *Container) initRepositories() {
	// --- IAM Repositories ---
	tenantRepo := tenantinfra.NewPostgresTenantRepository(c.DB)
	tenantConfigRepo := tenantinfra.NewPostgresTenantConfigRepository(c.DB)
	userRepo := userinfra.NewPostgresUserRepository(c.DB)
	userRoleRepo := userinfra.NewPostgresUserRoleRepository(c.DB)
	roleRepo := roleinfra.NewPostgresRoleRepository(c.DB)
	rolePermRepo := roleinfra.NewPostgresRolePermissionRepository(c.DB)
	tokenRepo := authinfra.NewPostgresTokenRepository(c.DB)
	sessionRepo := authinfra.NewPostgresSessionRepository(c.DB)
	invitationRepo := invitationinfra.NewPostgresInvitationRepository(c.DB)
	apiKeyRepo := apikeyinfra.NewPostgresAPIKeyRepository(c.DB)

	// --- Recruitment Repositories ---
	jobRepo := jobinfra.NewPostgresJobRepository(c.DB)
	candidateRepo := candidateinfra.NewPostgresCandidateRepository(c.DB)
	applicationRepo := applicationinfra.NewPostgresApplicationRepository(c.DB)

	// NOTE: Using a simple mock OTP repo or implementing one in Postgres would be needed here.
	// Assuming a Postgres implementation exists similar to others:
	// otpRepo := otpinfra.NewPostgresOTPRepository(c.DB)
	// For now, we assume it's handled within the OTP Service logic or passed as nil if not ready

	// --- Infrastructure Services ---
	stateManager := authinfra.NewRedisStateManager(c.Redis)
	passwordSvc := authinfra.NewBcryptPasswordService()

	// Token Service
	c.TokenService = auth.NewJWTService(
		c.AuthConfig.JWT.SecretKey,
		c.AuthConfig.JWT.AccessTokenTTL,
		c.AuthConfig.JWT.RefreshTokenTTL,
		c.AuthConfig.JWT.Issuer,
	)

	// --- Domain Services ---

	// 1. IAM Domain Services
	c.TenantService = tenantsrv.NewTenantService(tenantRepo, tenantConfigRepo, userRepo)
	c.UserService = usersrv.NewUserService(userRepo, userRoleRepo, tenantRepo, roleRepo, passwordSvc)
	c.RoleService = rolesrv.NewRoleService(roleRepo, rolePermRepo, tenantRepo)
	c.InvitationService = invitationsrv.NewInvitationService(invitationRepo, userRepo, tenantRepo, roleRepo)
	c.APIKeyService = apikeysrv.NewAPIKeyService(apiKeyRepo, tenantRepo, userRepo)

	// OAuth Services Map
	oauthServices := map[iam.OAuthProvider]auth.OAuthService{
		iam.OAuthProviderGoogle: auth.NewGoogleOAuthService(c.AuthConfig.OAuth.Google, stateManager),
		// Add Microsoft if configured
	}

	// Auth Handler (Core Logic)
	c.AuthService = auth.NewAuthHandlers(
		oauthServices,
		c.TokenService,
		userRepo,
		tenantRepo,
		tokenRepo,
		sessionRepo,
		stateManager,
		invitationRepo,
	)

	// 2. Recruitment Domain Services
	c.JobService = jobsrv.NewJobService(jobRepo, userRepo)
	c.CandidateService = candidatesrv.NewCandidateService(candidateRepo, userRepo)

	// Specialized Auth for Candidates
	candidateTokenSvc := candidateauth.NewCandidateTokenService(c.TokenService)

	// OTP Service (Needs a Notification Service impl, stubbing for now)

	otpRepo := otpinfra.NewPostgresOTPRepository(c.DB)
	c.OTPService = otpsrv.NewOTPService(otpRepo, NewConsoleNotifier())

	// Note: Since OTP dependencies weren't fully provided in the prompt,
	// we initialize CandidateAuthService with nil OTP service or require further implementation.
	// Assuming the services are correctly instantiated:
	c.CandidateAuthService = candidateauth.NewCandidateAuthService(candidateRepo, c.OTPService, candidateTokenSvc)

	c.ApplicationService = applicationsrv.NewApplicationService(
		applicationRepo,
		candidateRepo,
		jobRepo,
		userRepo,
		c.FileSystem,
	)

	// --- Handlers ---
	c.APIKeyHandlers = apikeyapi.NewAPIKeyHandlers(c.APIKeyService)
	c.InvitationHandlers = invitationapi.NewInvitationHandlers(c.InvitationService)
	c.JobHandlers = jobapi.NewHandlers(c.JobService)
	c.CandidateHandlers = candidateapi.NewHandlers(c.CandidateService)
	c.CandidateAuthHandlers = candidateauth.NewHandlers(c.CandidateAuthService, c.ApplicationService)
	c.ApplicationHandlers = applicationapi.NewHandlers(c.ApplicationService)

	// --- Middleware ---
	c.AuthMiddleware = auth.NewAuthMiddleware(c.TokenService)
	c.UnifiedAuthMiddleware = auth.NewAPIKeyMiddleware(c.APIKeyService, c.TokenService)
}

// ConsoleNotifier implements the NotificationService interface
// by printing OTP codes to the terminal/console
type ConsoleNotifier struct{}

// NewConsoleNotifier creates a new console-based OTP notifier
func NewConsoleNotifier() *ConsoleNotifier {
	return &ConsoleNotifier{}
}

// SendOTP prints the OTP code to the terminal
func (n *ConsoleNotifier) SendOTP(ctx context.Context, contact string, code string) error {
	fmt.Println("=", 50)
	fmt.Printf("📧 OTP NOTIFICATION\n")
	fmt.Printf("Contact: %s\n", contact)
	fmt.Printf("Code: %s\n", code)
	fmt.Println("=", 50)

	logx.Info(
		fmt.Sprintf("OTP sent to %s: %s", contact, code),
	)

	return nil
}
