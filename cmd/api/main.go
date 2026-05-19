package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SamvidPrasoon/EStore/internal/config"
	"github.com/SamvidPrasoon/EStore/internal/database"
	"github.com/SamvidPrasoon/EStore/internal/logger"
	"github.com/SamvidPrasoon/EStore/internal/server"
	"github.com/gin-gonic/gin"
)

func main() {
	//logger
	log := logger.New()
	//config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to DB")
	}
	//close DB connection
	DB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get DB connection client")
	}

	defer DB.Close()
	gin.SetMode(cfg.Server.GinMode)

	srv := server.New(cfg, db, log)
	router := srv.SetupRoutes()
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// for gracceful shutdown
	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("Starting http Server")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed to Start Server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to shutdown http Server")
	}
	log.Info().Msg("shutting down rest of the services")

}
