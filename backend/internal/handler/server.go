package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/na0chan-go/otayori-calendar/backend/internal/auth"
	"github.com/na0chan-go/otayori-calendar/backend/internal/config"
	"github.com/na0chan-go/otayori-calendar/backend/internal/service"
	"gorm.io/gorm"
)

type Server struct {
	*echo.Echo
	cfg       config.Config
	db        *gorm.DB
	sessions  auth.SessionManager
	tokens    auth.TokenCipher
	states    *auth.StateStore
	extractor *service.GeminiExtractor
}

func NewServer(cfg config.Config, db *gorm.DB) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{http.MethodGet, http.MethodPatch, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
	}))

	tokenCipher, err := auth.NewTokenCipher(cfg.SessionSecret)
	if err != nil {
		panic(err)
	}

	s := &Server{
		Echo:      e,
		cfg:       cfg,
		db:        db,
		sessions:  auth.NewSessionManager(cfg.SessionSecret),
		tokens:    tokenCipher,
		states:    auth.NewStateStore(10 * time.Minute),
		extractor: service.NewGeminiExtractor(cfg.GeminiAPIKey, cfg.GeminiModel, nil),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.GET("/healthz", s.healthz)
	s.GET("/auth/google/login", s.googleLogin)
	s.GET("/auth/google/callback", s.googleCallback)
	s.POST("/auth/logout", s.logout)
	s.GET("/api/auth/google/login", s.googleLogin)
	s.GET("/api/auth/google/callback", s.googleCallback)
	s.POST("/api/auth/logout", s.logout)
	s.GET("/api/me", s.me)
	s.POST("/api/manual-events", s.createManualEvent)
	s.POST("/api/manual-events/:id/retry", s.retryManualEvent)
	s.GET("/api/calendar-events", s.listCalendarEvents)
	s.POST("/api/letters", s.uploadLetter)
	s.GET("/api/letters", s.listLetters)
	s.GET("/api/letters/:id/image", s.showLetterImage)
	s.DELETE("/api/letters/:id", s.deleteLetter)
	s.POST("/api/letters/:id/extract-events", s.extractLetterEvents)
	s.GET("/api/extracted-events", s.listExtractedEvents)
	s.POST("/api/extracted-events/bulk-confirm", s.bulkConfirmExtractedEvents)
	s.POST("/api/extracted-events/bulk-ignore", s.bulkIgnoreExtractedEvents)
	s.POST("/api/extracted-events/bulk-register", s.bulkRegisterExtractedEvents)
	s.PATCH("/api/extracted-events/:id", s.updateExtractedEvent)
	s.POST("/api/extracted-events/:id/ignore", s.ignoreExtractedEvent)
	s.POST("/api/extracted-events/:id/register", s.registerExtractedEvent)
}

func (s *Server) healthz(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
