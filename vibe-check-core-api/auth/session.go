package auth

import (
	"log"
	"net/http"
	"time"
	"vibe/api"
	model "vibe/model/auth"
	"vibe/store"
)

func SetSession(w http.ResponseWriter, username string) {
	sessionToken := GenerateUUID()
	conn := store.Cache.Get()
	_, err := conn.Do("SETEX", sessionToken, "1800", username)
	defer conn.Close()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		Expires:  time.Now().Add(1800 * time.Second),
	})
}

func RemoveSession(w http.ResponseWriter, r *http.Request) {
	authStatus := &model.Auth{}
	authStatus.IsAuth = false
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			api.Respond(w, authStatus, http.StatusUnauthorized)
			return
		}
	}
	// remove session from cache
	conn := store.Cache.Get()
	_, err = conn.Do("DEL", string(c.Value))
	defer conn.Close()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// remove session from browser
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		SameSite: http.SameSiteDefaultMode,
		Expires:  time.Now(),
	})
	api.Respond(w, authStatus, http.StatusOK)
}

// Refresh session Token
func RefreshSession(w http.ResponseWriter, r *http.Request) {
	// TODO create refresh token method
}
