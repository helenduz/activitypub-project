package middlewares

import (
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

// Middleware for basic authentication
func BasicAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user, pass, ok := r.BasicAuth()
        if !ok || !authorize(user, pass) { // empty/invalid credentials
			// setting header which will prompt client for credentials
            w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func authorize(username, password string) bool {
	godotenv.Load()
    user := os.Getenv("ADMIN_USER")
    pass := os.Getenv("ADMIN_PASS")
    return username == user && password == pass
}
