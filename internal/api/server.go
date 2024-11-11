// internal/api/server.go

package api

import (
    "net/http"
    "web-service/internal/config"
    "web-service/internal/storage"
    "web-service/pkg/logging"
)

func NewServer(
    logger *logging.Logger,
    config *config.Config,
    commentStore *storage.CommentStore,
) http.Handler {
    mux := http.NewServeMux()

    // Add routes with all dependencies
    addRoutes(
        mux,
        logger,
        config,
        commentStore,
    )

    // Add middleware stack
    var handler http.Handler = mux
    handler = logging.NewLoggingMiddleware(logger, handler)

    // Create and apply auth middleware
    authMiddleware := newAuthMiddleware(config.JWTSecret)
    handler = authMiddleware(handler)

    // Create and apply CORS middleware
    corsMiddleware := newCORSMiddleware()
    handler = corsMiddleware(handler)

    return handler
}