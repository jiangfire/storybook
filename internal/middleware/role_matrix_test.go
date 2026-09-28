package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/model"
)

func TestRequireRouteRolesFollowsMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		routeKey string
		role     string
		want     int
	}{
		{"admin.ai", model.RoleAdmin, http.StatusOK},
		{"admin.ai", model.RoleTechLead, http.StatusForbidden},
		{"admin.ai", model.RoleProduct, http.StatusForbidden},
		{"admin.users", model.RoleAdmin, http.StatusOK},
		{"admin.users", model.RoleProduct, http.StatusForbidden},
		{"techlead", model.RoleTechLead, http.StatusOK},
		{"techlead", model.RoleAdmin, http.StatusOK},
		{"techlead", model.RoleDeveloper, http.StatusForbidden},
	}

	for _, tc := range cases {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(CtxRoleKey, tc.role)
		})
		r.GET("/x", RequireRouteRoles(tc.routeKey), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		if w.Code != tc.want {
			t.Fatalf("routeKey=%s role=%s: expected %d, got %d", tc.routeKey, tc.role, tc.want, w.Code)
		}
	}
}

func TestRequireRouteRolesPanicsOnUnknownKey(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unregistered route key")
		}
	}()
	RequireRouteRoles("not.registered")
}
