package initializers

import (
	"assignerPR/pkg/handlers"
	"assignerPR/pkg/pullrequest"
	"assignerPR/pkg/team"
	"assignerPR/pkg/user"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"log"
	"os"
)

func startGetEnv() {
	if os.Getenv("ENVIRONMENT") == "PROD" {
		return
	}

	err := godotenv.Load("local.env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

func startLogger() *zap.Logger {
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		log.Fatalf("LOG_LEVEL environment variable not set")
	}

	level, err := zapcore.ParseLevel(levelStr)
	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)

	zapLogger, err := config.Build()
	if err != nil {
		log.Fatalf("Error initializing zap logger: %v", err)
	}

	return zapLogger
}

func startPostgres() *gorm.DB {
	dsn := os.Getenv("PG_DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("Error initializing postgres: %v", err)
	}

	return db
}

func gormAutoMigrate(db *gorm.DB) {
	if os.Getenv("ENVIRONMENT") != "LOCAL" {
		return
	}

	if errAuto := db.AutoMigrate(
		&pullrequest.PullRequest{},
		&team.Team{},
		&user.User{},
	); errAuto != nil {
		log.Fatalf("AutoMigrate failed: %v", errAuto)
		return
	}
}

func initUserRoutes(router *gin.Engine, userHandler *handlers.UserHandler) {
	usersGroup := router.Group("/users")

	usersGroup.POST("/setIsActive", userHandler.SetIsActive)
	usersGroup.GET("/getReview", userHandler.GetUserReviews)
}

func initPullRequestRoutes(router *gin.Engine, pullRequestHandler *handlers.PullRequestHandler) {
	prsGroup := router.Group("/pullRequest")
	prsGroup.POST("/create", pullRequestHandler.CreatePR)
	prsGroup.POST("/merge", pullRequestHandler.Merge)
	prsGroup.POST("/reassign", pullRequestHandler.ReassignPR)
}

func initTeamRoutes(router *gin.Engine, teamHandler *handlers.TeamHandler) {
	teamsGroup := router.Group("/team")
	teamsGroup.POST("/add", teamHandler.AddTeam)
	teamsGroup.GET("/get", teamHandler.GetTeam)
}
