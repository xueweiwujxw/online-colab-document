package middleware

import (
	"context"
	"net/http"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/user"
)

type contextKey string

const currentUserKey contextKey = "current_user"

type Authenticator interface {
	CurrentUser(ctx context.Context, token string) (user.User, error)
}

func RequireAuth(service Authenticator, cookieName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		u, err := service.CurrentUser(r.Context(), cookie.Value)
		if err != nil {
			api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), currentUserKey, u)))
	})
}

func CurrentUser(ctx context.Context) (user.User, bool) {
	u, ok := ctx.Value(currentUserKey).(user.User)
	return u, ok
}
