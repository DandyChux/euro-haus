package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/dandychux/euro-haus/internal/models"
	"github.com/dandychux/euro-haus/internal/services"
)

type contextKey string

const userContextKey contextKey = "authenticated_user"

func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userContextKey).(*models.User)
	return user, ok
}

func tokenFromRequest(r *http.Request) (string, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", false
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}

func withAuthenticatedUser(
	r *http.Request,
	user *models.User,
) *http.Request {
	ctx := context.WithValue(
		r.Context(),
		userContextKey,
		user,
	)

	return r.WithContext(ctx)
}

// RequireAuth authenticates a regular user account.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := tokenFromRequest(r)
		if !ok {
			http.Error(
				w,
				"Missing or invalid authorization header",
				http.StatusUnauthorized,
			)
			return
		}

		authService := services.GetAuthService()
		user, err := authService.GetTokenUser(r.Context(), token)
		if err != nil {
			http.Error(
				w,
				"Invalid or expired token",
				http.StatusUnauthorized,
			)
			return
		}

		next.ServeHTTP(w, withAuthenticatedUser(r, user))
	})
}

// RequireAdminAuth authenticates an admin account.
func RequireAdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := tokenFromRequest(r)
		if !ok {
			http.Error(
				w,
				"Missing or invalid authorization header",
				http.StatusUnauthorized,
			)
			return
		}

		authService := services.GetAuthService()

		user, err := authService.GetTokenUser(r.Context(), token)
		if err != nil {
			http.Error(
				w,
				"Invalid or expired token",
				http.StatusUnauthorized,
			)
			return
		}

		if user.Role != "admin" {
			http.Error(
				w,
				"Administrator access required",
				http.StatusForbidden,
			)
			return
		}

		next.ServeHTTP(w, withAuthenticatedUser(r, user))
	})
}
