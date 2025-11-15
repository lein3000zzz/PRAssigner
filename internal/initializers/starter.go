package initializers

import (
	"assignerPR/pkg/handlers"
	"assignerPR/pkg/pullrequest"
	"assignerPR/pkg/team"
	"assignerPR/pkg/user"
	"log"
	"os"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RunPRAssigner() {
	startGetEnv()

	zapLogger := startLogger()

	defer func(zapLogger *zap.Logger) {
		err := zapLogger.Sync()
		if err != nil {
			log.Fatal("Error syncing zap logger:", err)
		}
	}(zapLogger)

	logger := zapLogger.Sugar()
	db := startPostgres()

	gormAutoMigrate(db)

	userRepo := user.NewUsersRepoPg(logger, db)
	teamRepo := team.NewTeamsRepoPg(logger, db)
	prRepo := pullrequest.NewPullRequestsRepoPg(logger, db)

	userHandler := handlers.NewUserHandler(logger, userRepo, prRepo)
	teamHandler := handlers.NewTeamHandler(logger, teamRepo, prRepo)
	prHandler := handlers.NewPullRequestHandler(logger, prRepo)

	router := gin.New()
	router.Use(ginzap.Ginzap(zapLogger, time.RFC3339, true))

	router.Use(ginzap.RecoveryWithZap(zapLogger, true))

	initUserRoutes(router, userHandler)
	initTeamRoutes(router, teamHandler)
	initPullRequestRoutes(router, prHandler)

	logger.Fatal(router.Run(":" + os.Getenv("PORT")))
}
