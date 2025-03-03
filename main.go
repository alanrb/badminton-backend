package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/alanrb/badminton/backend/database"
	"github.com/alanrb/badminton/backend/handlers"
	"github.com/alanrb/badminton/backend/middleware"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	echoadapter "github.com/awslabs/aws-lambda-go-api-proxy/echo"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

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
		protected = e.Group("/api", middleware.JWTConfig(jwtSecret))
	} else {
		protected = e.Group("/api")
		protected.PUT("/auth/cognito", handlers.HandleCognitoUser)
	}

	protected.POST("/sessions", handlers.CreateSession)
	protected.GET("/sessions", handlers.GetSessions)
	protected.GET("/sessions/:session_id", handlers.GetSessionDetails)
	protected.PUT("/sessions/:session_id", handlers.UpdateSession)
	protected.DELETE("/sessions/:session_id", handlers.DeleteSession)
	protected.DELETE("/sessions/:session_id/attend", handlers.CancelAttendance)
	protected.POST("/sessions/:session_id/attend", handlers.AttendSession)

	protected.GET("/profile", handlers.GetProfile)

	protected.GET("/users/attended-sessions", handlers.GetAttendedSessions)

	protected.GET("/badminton-courts", handlers.GetBadmintonCourts)
	protected.GET("/badminton-courts/:id", handlers.GetBadmintonCourt)

	// Admin routes
	adminGroup := protected.Group("/admin", middleware.AdminOnly)
	adminGroup.POST("/users", handlers.CreateUser)
	adminGroup.GET("/users", handlers.GetUsers)

	adminGroup.POST("/sessions", handlers.CreateSession)
	adminGroup.PUT("/sessions/:id/status", handlers.UpdateSessionStatus)
	adminGroup.DELETE("/sessions/:id", handlers.DeleteSession)

	adminGroup.GET("/badminton-courts", handlers.GetBadmintonCourts)
	adminGroup.GET("/badminton-courts/:id", handlers.GetBadmintonCourt)
	adminGroup.POST("/badminton-courts", handlers.CreateBadmintonCourt)
	adminGroup.PUT("/badminton-courts/:id", handlers.UpdateBadmintonCourt)
	adminGroup.DELETE("/badminton-courts/:id", handlers.DeleteBadmintonCourt)

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
