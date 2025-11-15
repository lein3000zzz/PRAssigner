package initializers

import (
	"assignerPR/pkg/handlers"
	"assignerPR/pkg/pullrequest"
	"assignerPR/pkg/team"
	"assignerPR/pkg/user"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	router.Use(ginzap.GinzapWithConfig(zapLogger, &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		Skipper: func(c *gin.Context) bool {
			return c.Request.URL.Path == "/metrics" && c.Request.Method == "GET"
		},
	}))

	router.Use(ginzap.RecoveryWithZap(zapLogger, true))

	initUserRoutes(router, userHandler)
	initTeamRoutes(router, teamHandler)
	initPullRequestRoutes(router, prHandler)
	initMetrics(router)
	initpprof(router)

	srv := &http.Server{
		Addr:    ":" + os.Getenv("PORT"),
		Handler: router,
	}

	go func() {
		logger.Info("Starting server on port " + os.Getenv("PORT"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down the server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("the serrver was forced to shutdown:", err)
	}

	logger.Info("Server exited")
}
