package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/na0chan-go/otayori-calendar/backend/internal/auth"
	"github.com/na0chan-go/otayori-calendar/backend/internal/config"
	"gorm.io/gorm"
)

type Server struct {
	*echo.Echo
	cfg      config.Config
	db       *gorm.DB
	sessions auth.SessionManager
	tokens   auth.TokenCipher
	states   *auth.StateStore
}

func NewServer(cfg config.Config, db *gorm.DB) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
	}))

	tokenCipher, err := auth.NewTokenCipher(cfg.SessionSecret)
	if err != nil {
		panic(err)
	}

	s := &Server{
		Echo:     e,
		cfg:      cfg,
		db:       db,
		sessions: auth.NewSessionManager(cfg.SessionSecret),
		tokens:   tokenCipher,
		states:   auth.NewStateStore(10 * time.Minute),
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
}

func (s *Server) healthz(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
