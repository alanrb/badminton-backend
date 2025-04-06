package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/alanrb/badminton/backend/database"
	"github.com/alanrb/badminton/backend/handlers"
	"github.com/alanrb/badminton/backend/middleware"
	"github.com/alanrb/badminton/backend/rbac"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	echoadapter "github.com/awslabs/aws-lambda-go-api-proxy/echo"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// EchoLambdaV2 is the adapter for AWS Lambda
var echoLambda *echoadapter.EchoLambdaV2

func loadEnv() {
	// Check if running in AWS Lambda
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
		// Not in Lambda, load .env file
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found or error loading .env file")
		}
	}
}

func main() {
	// Load environment variables
	loadEnv()

	debugMode := os.Getenv("DEBUG_MODE") == "true"

	// Initialize database
	database.Init(os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSL_MODE"), debugMode)

	// Create Echo instance
	e := echo.New()
	e.Server.ReadHeaderTimeout = time.Duration(10) * time.Second
	e.Server.ReadTimeout = time.Duration(30) * time.Second
	e.Server.WriteTimeout = time.Duration(60) * time.Second
	e.Debug = debugMode

	// Initialize OAuth2 configuration
	oauth2 := oauth2.Config{
		RedirectURL:  os.Getenv("AUTH_REDIRECT_URL"),
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))

	// Middleware
	e.Use(echomiddleware.ContextTimeoutWithConfig(echomiddleware.ContextTimeoutConfig{
		Timeout: 29 * time.Second,
	}))
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.Context)

	// Public routes
	e.GET("/auth/google/login", func(c echo.Context) error {
		return handlers.HandleGoogleLogin(c, &oauth2)
	})
	e.GET("/auth/google/callback", func(c echo.Context) error {
		return handlers.HandleGoogleCallback(c, jwtSecret, os.Getenv("CMS_URL"), &oauth2)
	})

	// Protected routes
	var protected *echo.Group
	if os.Getenv("COGNITO_ISSUER") == "" {
		// If Cognito is not used, use JWT middleware
		protected = e.Group("/api", middleware.JWTConfig(jwtSecret))
	} else {
		// If Cognito is used, use Cognito middleware
		protected = e.Group("/api")
		protected.PUT("/auth/cognito", handlers.HandleCognitoUser)
	}

	protected.POST("/sessions", handlers.CreateSession, middleware.RBAC(database.DB, string(rbac.PermissionCreateSessions)))
	protected.GET("/sessions", handlers.GetSessions)
	protected.GET("/sessions/:session_id", handlers.GetSessionDetails)
	protected.PUT("/sessions/:session_id", handlers.UpdateSession, middleware.RBAC(database.DB, string(rbac.PermissionCreateSessions)))
	protected.DELETE("/sessions/:session_id", handlers.DeleteSession, middleware.RBAC(database.DB, string(rbac.PermissionDeleteSessions)))
	protected.DELETE("/sessions/:session_id/attend", handlers.CancelAttendance)
	protected.POST("/sessions/:session_id/attend", handlers.AttendSession)

	protected.GET("/profile", handlers.GetProfile)

	protected.GET("/users/attended-sessions", handlers.GetAttendedSessions)

	protected.GET("/groups", handlers.ListGroups)
	protected.POST("/groups", handlers.CreateGroup, middleware.RBAC(database.DB, string(rbac.PermissionCreateGroups)))
	protected.POST("/groups/:group_id/players", handlers.AddPlayerToGroup, middleware.RBAC(database.DB, string(rbac.PermissionAddGroupPlayer)))
	protected.DELETE("/groups/:group_id", handlers.DeleteGroup, middleware.RBAC(database.DB, string(rbac.PermissionDeleteGroups)))
	protected.GET("/groups/:group_id", handlers.GetGroupDetails)

	protected.GET("/courts", handlers.GetBadmintonCourts)
	protected.GET("/courts/:id", handlers.GetBadmintonCourt)

	// Admin routes (only accessible to admins)
	adminGroup := protected.Group("/admin", middleware.AdminOnly)
	adminGroup.POST("/users", handlers.CreateUser, middleware.RBAC(database.DB, string(rbac.PermissionListUsers)))
	adminGroup.GET("/users/:user_id", handlers.GetUser, middleware.RBAC(database.DB, string(rbac.PermissionEditUsers)))
	adminGroup.GET("/users", handlers.GetUsers, middleware.RBAC(database.DB, string(rbac.PermissionListUsers)))
	adminGroup.PUT("/users/:user_id", handlers.UpdateUser, middleware.RBAC(database.DB, string(rbac.PermissionEditUsers)))

	adminGroup.POST("/sessions", handlers.CreateSession, middleware.RBAC(database.DB, string(rbac.PermissionCreateSessions)))
	adminGroup.PUT("/sessions/:id/status", handlers.UpdateSessionStatus, middleware.RBAC(database.DB, string(rbac.PermissionCreateSessions)))
	adminGroup.DELETE("/sessions/:id", handlers.DeleteSession, middleware.RBAC(database.DB, string(rbac.PermissionDeleteSessions)))

	adminGroup.POST("/courts", handlers.CreateBadmintonCourt, middleware.RBAC(database.DB, string(rbac.PermissionCreateCourts)))
	adminGroup.PUT("/courts/:id", handlers.UpdateBadmintonCourt, middleware.RBAC(database.DB, string(rbac.PermissionEditCourts)))
	adminGroup.DELETE("/courts/:id", handlers.DeleteBadmintonCourt, middleware.RBAC(database.DB, string(rbac.PermissionDeleteCourts)))

	// Check if running in Lambda
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		// Initialize EchoLambda adapter
		echoLambda = echoadapter.NewV2(e)
		lambda.Start(Handler)
	} else {
		// Start local server
		e.Logger.Fatal(e.Start(":8080"))
	}
}

// Handler processes Lambda events
func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return echoLambda.ProxyWithContext(ctx, req)
}
