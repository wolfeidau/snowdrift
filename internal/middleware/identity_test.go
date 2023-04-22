package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestIdentity(t *testing.T) {
	assert := require.New(t)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	config := IdentityConfig{
		Name:     "test",
		SameSite: http.SameSiteLaxMode,
		TTLDays:  30,
		Path:     "/",
		Domain:   "example.com",
	}

	h := Identity(config)(func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	})

	err := h(c)
	assert.NoError(err)

	assert.Equal(http.StatusOK, rec.Code)
	assert.Equal("test", rec.Body.String())

	cookie := rec.Result().Cookies()[0]
	assert.Equal("test", cookie.Name)
	assert.NotEmpty(cookie.Value)
	// assert.Equal(t, 30*24*time.Hour, cookie.Expires.Sub(time.Now()))
	assert.Equal("/", cookie.Path)
	assert.Equal("example.com", cookie.Domain)
	assert.Equal(http.SameSiteLaxMode, cookie.SameSite)
}
