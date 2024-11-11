// internal/api/routes.go

package api

import (
	"net/http"
	"time"
	"web-service/internal/auth"
	"web-service/internal/config"
	"web-service/internal/storage"
	"web-service/pkg/logging"
)

func addRoutes(
    mux *http.ServeMux,
    logger *logging.Logger,
    config *config.Config,
    commentStore *storage.CommentStore,
) {
    jwtManager := auth.NewJWTManager(config.JWTSecret, 24*time.Hour)

    mux.Handle("/api/v1/login", handleLogin(logger, jwtManager))
    mux.Handle("/api/v1/comments", handleComments(logger, commentStore))
    mux.Handle("/api/v1/comments/", handleComment(logger, commentStore))
    mux.Handle("/healthz", handleHealthz(logger))
    mux.Handle("/", http.NotFoundHandler())
}