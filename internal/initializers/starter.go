package initializers

import (
	handlers2 "assignerPR/internal/handlers"
	"assignerPR/internal/pullrequest"
	"assignerPR/pkg/team"
	"assignerPR/pkg/user"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	shutdownContextTimeout = 30 * time.Second
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

	userHandler := handlers2.NewUserHandler(logger, userRepo, prRepo)
	teamHandler := handlers2.NewTeamHandler(logger, teamRepo, prRepo)
	prHandler := handlers2.NewPullRequestHandler(logger, prRepo)

	router := gin.New()
	initMetricsMdlwr(router)

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
	metricsSrv := initMetricsServer()
	initpprof(router)

	srv := &http.Server{
		Addr:    ":" + os.Getenv("PORT"),
		Handler: router,
	}

	go func() {
		logger.Info("Starting main server on port " + os.Getenv("PORT"))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("listen: %s\n", err)
		}
	}()

	go func() {
		logger.Info("Starting metrics server on port " + os.Getenv("METRICS_PORT"))
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down the server")

	numberOfServerGoroutines := 2
	wg := &sync.WaitGroup{}

	wg.Add(numberOfServerGoroutines)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownContextTimeout)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Fatal("the serrver was forced to shutdown:", err)
		}
		wg.Done()
	}()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownContextTimeout)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Fatal("the serrver was forced to shutdown:", err)
		}
		wg.Done()
	}()

	wg.Wait()

	logger.Info("Server exited")
}
