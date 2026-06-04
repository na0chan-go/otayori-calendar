package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/na0chan-go/otayori-calendar/backend/internal/auth"
	"github.com/na0chan-go/otayori-calendar/backend/internal/model"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type googleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (s *Server) googleLogin(c echo.Context) error {
	oauthConfig, err := s.cfg.GoogleOAuthConfig()
	if err != nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, err.Error())
	}

	state, err := s.states.Create(time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create oauth state")
	}

	url := oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func (s *Server) googleCallback(c echo.Context) error {
	if !s.states.Consume(c.QueryParam("state"), time.Now()) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid oauth state")
	}

	code := c.QueryParam("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing oauth code")
	}

	oauthConfig, err := s.cfg.GoogleOAuthConfig()
	if err != nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, err.Error())
	}

	ctx := c.Request().Context()
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "failed to exchange oauth code")
	}

	info, err := fetchGoogleUserInfo(ctx, oauthConfig.Client(ctx, token))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "failed to fetch google user info")
	}

	user, err := s.saveLogin(ctx, info, token)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save login")
	}

	s.setSessionCookie(c, user.ID)
	return c.Redirect(http.StatusTemporaryRedirect, s.cfg.FrontendURL+"/")
}

func (s *Server) logout(c echo.Context) error {
	c.SetCookie(&http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.IsProduction(),
	})
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) me(c echo.Context) error {
	userID, err := s.currentUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
	}

	var user model.User
	if err := s.db.WithContext(c.Request().Context()).First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load user")
	}

	return c.JSON(http.StatusOK, map[string]any{"user": user})
}

func (s *Server) saveLogin(ctx context.Context, info googleUserInfo, token *oauth2.Token) (model.User, error) {
	var user model.User
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "google_user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"email", "name"}),
		}).Create(&model.User{
			ID:           uuid.New(),
			GoogleUserID: info.ID,
			Email:        info.Email,
			Name:         info.Name,
		}).Error; err != nil {
			return err
		}

		if err := tx.Where("google_user_id = ?", info.ID).First(&user).Error; err != nil {
			return err
		}

		encryptedRefreshToken := ""
		if token.RefreshToken != "" {
			var err error
			encryptedRefreshToken, err = s.tokens.Encrypt(token.RefreshToken)
			if err != nil {
				return err
			}
		} else {
			var existing model.GoogleToken
			if err := tx.Select("refresh_token").First(&existing, "user_id = ?", user.ID).Error; err == nil {
				encryptedRefreshToken = existing.RefreshToken
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		accessToken, err := s.tokens.Encrypt(token.AccessToken)
		if err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"access_token":  accessToken,
				"refresh_token": encryptedRefreshToken,
				"expiry":        token.Expiry,
				"updated_at":    time.Now(),
			}),
		}).Create(&model.GoogleToken{
			UserID:       user.ID,
			AccessToken:  accessToken,
			RefreshToken: encryptedRefreshToken,
			Expiry:       token.Expiry,
		}).Error
	})

	return user, err
}

func (s *Server) setSessionCookie(c echo.Context, userID uuid.UUID) {
	c.SetCookie(&http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    s.sessions.Sign(userID, time.Now()),
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.IsProduction(),
	})
}

func (s *Server) currentUserID(c echo.Context) (uuid.UUID, error) {
	cookie, err := c.Cookie(auth.SessionCookieName)
	if err != nil {
		return uuid.Nil, err
	}
	return s.sessions.Verify(cookie.Value, time.Now())
}

func fetchGoogleUserInfo(ctx context.Context, client *http.Client) (googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return googleUserInfo{}, err
	}

	res, err := client.Do(req)
	if err != nil {
		return googleUserInfo{}, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return googleUserInfo{}, errors.New("google userinfo returned non-2xx")
	}

	var info googleUserInfo
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return googleUserInfo{}, err
	}
	if info.ID == "" || info.Email == "" {
		return googleUserInfo{}, errors.New("google userinfo missing id or email")
	}

	return info, nil
}
