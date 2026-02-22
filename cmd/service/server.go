package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/donder-core/hiroba-scraper-service/internal/api"
	"github.com/donder-core/hiroba-scraper-service/internal/auth"
	"github.com/donder-core/hiroba-scraper-service/internal/scraper"
	"github.com/donder-core/hiroba-scraper-service/internal/service"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const reauthInterval = 2 * time.Minute
const port = ":6614"

func init() {
	_ = godotenv.Load()
}

func main() {
	logger := log.New(os.Stdout, "[MAIN] ", log.LstdFlags)

	tokenService := auth.GetInstance()

	taikoUsername := os.Getenv("TAIKO_USERNAME")
	taikoPassword := os.Getenv("TAIKO_PASSWORD")

	if taikoUsername == "" || taikoPassword == "" {
		logger.Fatalf("TAIKO_USERNAME or TAIKO_PASSWORD is not set")
	}

	// Initial authentication
	if _, err := tokenService.Authenticate(taikoUsername, taikoPassword); err != nil {
		logger.Fatalf("Initial authentication failed: %v", err)
	}
	logger.Println("Initial authentication successful")

	// Setup graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start periodic reauthentication in background with graceful shutdown support
	go func() {
		logger.Printf("Starting reauthentication background task (interval: %v)", reauthInterval)
		ticker := time.NewTicker(reauthInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				logger.Println("Reauthenticating...")
				if _, err := tokenService.Authenticate(taikoUsername, taikoPassword); err != nil {
					logger.Printf("Reauthentication failed: %v", err)
				} else {
					logger.Println("Reauthentication successful")
				}
			case <-ctx.Done():
				logger.Println("Stopping reauthentication background task...")
				return
			}
		}
	}()

	tokenHandler := api.NewTokenHandler(tokenService)
	htmlScraper := scraper.NewHtmlScraper(tokenService.GetClient())
	scoreService := service.NewScoreService(htmlScraper, tokenHandler)

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogLatency:       true,
		LogRemoteIP:      false,
		LogHost:          false,
		LogMethod:        true,
		LogURI:           true,
		LogRequestID:     true,
		LogUserAgent:     false, // Disabled user agent logging
		LogStatus:        true,
		LogContentLength: false,
		LogResponseSize:  false,
		HandleError:      true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			logger := c.Logger()
			if v.Error == nil {
				logger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
					slog.String("method", v.Method),
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.Duration("latency", v.Latency),
					slog.String("host", v.Host),
					slog.String("bytes_in", v.ContentLength),
					slog.Int64("bytes_out", v.ResponseSize),
					slog.String("remote_ip", v.RemoteIP),
					slog.String("request_id", v.RequestID),
				)
				return nil
			}

			logger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.Duration("latency", v.Latency),
				slog.String("host", v.Host),
				slog.String("bytes_in", v.ContentLength),
				slog.Int64("bytes_out", v.ResponseSize),
				slog.String("remote_ip", v.RemoteIP),
				slog.String("request_id", v.RequestID),
				slog.String("error", v.Error.Error()),
			)
			return nil
		},
	}))

	if err := api.SetupRoutes(e, tokenHandler, scoreService); err != nil {
		e.Logger.Error("failed to setup routes", "error", err)
		return
	}

	logger.Printf("Starting server on %s", port)
	sc := echo.StartConfig{
		Address:         port,
		GracefulTimeout: 5 * time.Second,
	}

	if err := sc.Start(ctx, e); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}

	logger.Println("Server shutdown complete")
}
