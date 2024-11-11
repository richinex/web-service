// internal/api/middleware.go

package api

import (
    "context"
    "net/http"
    "strings"
    "time"
    "web-service/internal/auth"
)

type contextKey string

const (
    UserIDKey contextKey = "user_id"
    UserRoleKey contextKey = "user_role"
)

func newAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
    jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip auth for health check and other public endpoints
            if r.URL.Path == "/healthz" || r.URL.Path == "/api/v1/login" {
                next.ServeHTTP(w, r)
                return
            }

            authHeader := r.Header.Get("Authorization")
            if !strings.HasPrefix(authHeader, "Bearer ") {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
            claims, err := jwtManager.ValidateToken(tokenStr)
            if err != nil {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }

            // Add user info to context
            ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
            ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func newCORSMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Access-Control-Allow-Origin", "*")
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// Helper functions to extract user info from context
func UserIDFromContext(ctx context.Context) string {
    if id, ok := ctx.Value(UserIDKey).(string); ok {
        return id
    }
    return ""
}

func UserRoleFromContext(ctx context.Context) string {
    if role, ok := ctx.Value(UserRoleKey).(string); ok {
        return role
    }
    return ""
}