package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
)

type IdentityConfig struct {
	Name     string
	SameSite http.SameSite
	TTLDays  int
	Path     string
	Domain   string
	Skipper  middleware.Skipper
}

func Identity(config IdentityConfig) echo.MiddlewareFunc {
	if config.Skipper == nil {
		config.Skipper = middleware.DefaultSkipper
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			log.Ctx(c.Request().Context()).Debug().Msg("identity cookie")
			ck, err := getOrCreateCookie(config, c)
			if err != nil {
				log.Ctx(c.Request().Context()).Warn().Err(err).Msg("create cookie failed")
				return c.String(http.StatusInternalServerError, "server error")
			}

			c.SetCookie(ck)

			return next(c)
		}
	}
}

func getOrCreateCookie(config IdentityConfig, c echo.Context) (*http.Cookie, error) {
	ctx := c.Request().Context()

	existingCookie, err := c.Cookie(config.Name)
	if err == http.ErrNoCookie {
		newID := uuid.NewString()
		log.Ctx(ctx).Debug().Str("id", newID).Msg("create identity cookie")
		return createCookie(config, newID), nil
	}
	if err != nil {
		return nil, err
	}

	log.Ctx(ctx).Debug().Str("id", existingCookie.Value).Msg("renew identity cookie")
	return createCookie(config, existingCookie.Value), nil
}

func createCookie(config IdentityConfig, v string) *http.Cookie {
	return &http.Cookie{
		Name:     config.Name,
		Value:    v,
		Expires:  time.Now().Add(time.Duration(config.TTLDays) * 24 * time.Hour),
		Path:     config.Path,
		Domain:   config.Domain,
		SameSite: config.SameSite,
	}
}
