package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// BasicAuth gates an endpoint behind a fixed username/password pair using
// constant-time comparison so timing oracles cannot recover the secret.
// The handler aborts with 401 and the standard WWW-Authenticate header on
// mismatch, which lets `curl -u` and Prometheus scrape configs work without
// extra hand-holding.
func BasicAuth(user, pass, realm string) gin.HandlerFunc {
	if realm == "" {
		realm = "metrics"
	}
	expectedUser := []byte(user)
	expectedPass := []byte(pass)
	wwwAuth := `Basic realm="` + realm + `"`

	return func(c *gin.Context) {
		gotUser, gotPass, ok := c.Request.BasicAuth()
		if !ok {
			c.Header("WWW-Authenticate", wwwAuth)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		userMatch := subtle.ConstantTimeCompare([]byte(gotUser), expectedUser) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(gotPass), expectedPass) == 1
		if !(userMatch && passMatch) {
			c.Header("WWW-Authenticate", wwwAuth)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	}
}
