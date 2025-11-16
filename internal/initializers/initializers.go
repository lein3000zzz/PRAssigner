package initializers

import (
	handlers2 "assignerPR/internal/handlers"
	"assignerPR/internal/handlers/mdlwr"
	"assignerPR/internal/metrics"
	"assignerPR/internal/pullrequest"
	"assignerPR/pkg/team"
	"assignerPR/pkg/user"
	"net/http"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"log"
	"os"
)

// При запуске без докера. Комментить код и не удалять - плохо, но тут есть причины
// func startGetEnv() {
//	if os.Getenv("ENVIRONMENT") == "PROD" {
//		return
//	}
//
//	err := godotenv.Load("local.env")
//
//	if err != nil {
//		log.Fatalf("Error loading .env file")
//	}
// }

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

func initUserRoutes(router *gin.Engine, userHandler *handlers2.UserHandler) {
	usersGroup := router.Group("/users")

	auth := initAdminAuthMdlwr()
	usersGroup.POST("/setIsActive", auth.MiddlewareFunc(), userHandler.SetIsActive)
	usersGroup.POST("/deactivateTeam", auth.MiddlewareFunc(), userHandler.DeactivateTeam)
	usersGroup.GET("/getReview", userHandler.GetUserReviews)
}

func initPullRequestRoutes(router *gin.Engine, pullRequestHandler *handlers2.PullRequestHandler) {
	prsGroup := router.Group("/pullRequest")

	prsGroup.POST("/create", pullRequestHandler.CreatePR)
	prsGroup.POST("/merge", pullRequestHandler.Merge)
	prsGroup.POST("/reassign", pullRequestHandler.ReassignPR)
}

func initTeamRoutes(router *gin.Engine, teamHandler *handlers2.TeamHandler) {
	teamsGroup := router.Group("/team")

	teamsGroup.POST("/add", teamHandler.AddTeam)
	teamsGroup.GET("/get", teamHandler.GetTeam)
	teamsGroup.GET("/pr-stats", teamHandler.StatsTeam)
}

func initpprof(router *gin.Engine) {
	if os.Getenv("ENVIRONMENT") == "LOCAL" {
		pprof.Register(router)
	}
}

func initMetricsMdlwr(router *gin.Engine) {
	router.Use(metrics.GinMiddleware)
}

// Отдельный сервер, чтобы скрыть чувствительную информацию в виде метрик от лишних глаз
func initMetricsServer() *http.Server {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/metrics", metrics.Handler())

	srv := &http.Server{
		Addr:    ":" + os.Getenv("METRICS_PORT"),
		Handler: r,
	}

	return srv
}

// В соответствии со спецификацией openapi.yml
func initAdminAuthMdlwr() *jwt.GinJWTMiddleware {
	adminSecret := os.Getenv("ADMIN_JWT_SECRET")

	if adminSecret == "" {
		log.Fatalf("ADMIN_JWT_SECRET environment variable not set")
	}

	auth, err := mdlwr.GetAdminAuthMiddleware(adminSecret)
	if err != nil {
		log.Fatal("JWT Error: " + err.Error())
	}

	return auth
}
