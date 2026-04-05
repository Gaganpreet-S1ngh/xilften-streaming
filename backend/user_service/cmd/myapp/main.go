package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"syscall"
	"time"

	"github.com/Gaganpreet-S1ngh/xilften-user-service/internal/platform/config"
	"github.com/Gaganpreet-S1ngh/xilften-user-service/internal/platform/database"
	"github.com/Gaganpreet-S1ngh/xilften-user-service/internal/platform/httpserver"
	"github.com/Gaganpreet-S1ngh/xilften-user-service/internal/user"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	/* INITIALIZE ENV AND CONTEXTS */
	cfg := config.LoadConfig()
	rootCtx, rootCancel := context.WithCancel(context.Background())

	/* INITIALIZE SERVICES */

	// DATABASE (SQL)
	db := database.NewSQLDatabase(cfg.Logger)

	if err := db.Connect(rootCtx, cfg.DatabaseDSN); err != nil {
		log.Fatal("Database connection failed!", zap.Error(err))
	}

	if err := db.RegisterModels(rootCtx,
		(*user.UserGenre)(nil), (*user.User)(nil),
		(*user.Genre)(nil)); err != nil {
		cfg.Logger.Warn("Error creating tables!", zap.Error(err))
	}

	// DATABASE (REDIS)
	redisDB := database.NewRedisDatabase(cfg.Logger)

	if err := redisDB.Connect(rootCtx, cfg.RedisDSN); err != nil {
		log.Fatal("Redis Database connection failed!", zap.Error(err))
	}

	if err := redisDB.Ping(rootCtx); err != nil {
		log.Println(err)
	}

	// DEPENDENCY INJECTIONS
	repository := user.NewRepository(db.GetDBClient(), cfg.Logger)
	service := user.NewService(repository, cfg.Logger)
	handler := user.NewHandler(service)

	// GIN ENGINE
	gin.SetMode(cfg.GinMode)
	ginEngine := httpserver.NewGinEngine(cfg.Logger)
	routes := user.NewRoutes(ginEngine, handler)

	/* INITIALIZE ROUTES */

	routes.SetupPublicRoutes()

	/* START SERVERS */

	httpServer := httpserver.NewHTTPServer(cfg.Port, ginEngine)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cfg.Logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	/* SHUTDOWN MECHANISM */

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	cfg.Logger.Info("Shutdown Signal Recieved!")

	// Stop Server
	shutDownHTTP(httpServer)

	// Stop Workers
	rootCancel()

	// Close Services
	db.GetDBClient().Close()
	redisDB.Close()
	cfg.Logger.Sync()

}

func shutDownHTTP(s *http.Server) {
	if s != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := s.Shutdown(ctx)

		if err != nil {
			log.Printf("Shutdown Error: %v", err)
			return
		}
	}
	log.Println("Server gracefully shutdown")
}
