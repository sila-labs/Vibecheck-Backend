package auth

import (
	"fmt"
	"net/http"
	"vibe/api"
	model "vibe/model/auth"
	"vibe/store"

	"github.com/google/uuid"
)

func GenerateUUID() string {
	return uuid.NewString()
}

// Middleware for validating authentication for API access
func RequireAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authStatus := &model.Auth{}
		authStatus.IsAuth = false
		c, err := r.Cookie("session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				api.Respond(w, authStatus, http.StatusUnauthorized)
				return
			}
			api.Respond(w, authStatus, http.StatusBadRequest)
			return
		}
		sessionToken := c.Value

		conn := store.Cache.Get()
		res, err := conn.Do("GET", sessionToken)
		defer conn.Close()
		if err != nil {
			api.Respond(w, authStatus, http.StatusInternalServerError)
			return
		}
		if res == nil {
			api.Respond(w, authStatus, http.StatusUnauthorized)
			return
		}
		//authStatus.IsAuth = false
		//ctx := context.WithValue(r.Context(), "is_auth", authStatus.IsAuth)
		//next.ServeHTTP(w, r.WithContext(ctx))
		next.ServeHTTP(w, r)

	})
}

func IsAuthenticated(w http.ResponseWriter, r *http.Request) {
	authStatus := &model.Auth{}
	authStatus.IsAuth = false
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			api.Respond(w, authStatus, http.StatusUnauthorized)
			return
		}
		api.Respond(w, authStatus, http.StatusBadRequest)
		return
	}
	sessionToken := c.Value
	fmt.Println(sessionToken)
	conn := store.Cache.Get()
	res, err := conn.Do("GET", sessionToken)
	defer conn.Close()
	if err != nil {
		api.Respond(w, authStatus, http.StatusInternalServerError)
		return
	}
	if res == nil {
		api.Respond(w, authStatus, http.StatusUnauthorized)
		return
	}
	authStatus.IsAuth = true
	api.Respond(w, authStatus, http.StatusOK)
}
