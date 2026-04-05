package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"syscall"
	"time"

	"github.com/Gaganpreet-S1ngh/xilften-streaming/internal/platform/config"
	"github.com/Gaganpreet-S1ngh/xilften-streaming/internal/platform/database"
	"github.com/Gaganpreet-S1ngh/xilften-streaming/internal/platform/httpserver"
	"github.com/Gaganpreet-S1ngh/xilften-streaming/internal/streaming"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	/* INITIALIZE ENV AND CONTEXTS */
	cfg := config.LoadConfig()
	rootCtx, rootCancel := context.WithCancel(context.Background())

	/* INITIALIZE SERVICES */

	// DATABASE
	db := database.NewSQLDatabase(cfg.Logger)

	if err := db.Connect(rootCtx, cfg.DatabaseDSN); err != nil {
		log.Fatal("Database connection failed!", zap.Error(err))
	}

	if err := db.RegisterModels(rootCtx,
		(*streaming.MovieGenre)(nil), (*streaming.Movie)(nil),
		(*streaming.Genre)(nil)); err != nil {
		cfg.Logger.Warn("Error creating tables!", zap.Error(err))
	}

	// DEPENDENCY INJECTIONS
	handler := streaming.NewHandler()

	// GIN ENGINE
	gin.SetMode(cfg.GinMode)
	ginEngine := httpserver.NewGinEngine(cfg.Logger)
	routes := streaming.NewRoutes(ginEngine, handler)

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
