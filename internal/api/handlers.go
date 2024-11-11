// internal/api/handlers.go

package api

import (
    "context"
    "net/http"
    "strings"
    "time"
    "web-service/internal/storage"
    "web-service/internal/auth"
    "web-service/pkg/logging"
)

// Request/response types
type createCommentRequest struct {
    Content string `json:"content"`
    Author  string `json:"author"`
}

type commentResponse struct {
    ID        string    `json:"id"`
    Content   string    `json:"content"`
    Author    string    `json:"author"`
    CreatedAt time.Time `json:"created_at"`
    UserID    string    `json:"user_id,omitempty"`
}

// Validator implementation
func (r createCommentRequest) Valid(ctx context.Context) map[string]string {
    problems := make(map[string]string)
    if strings.TrimSpace(r.Content) == "" {
        problems["content"] = "content is required"
    }
    if len(r.Content) > 1000 {
        problems["content"] = "content must be less than 1000 characters"
    }
    if strings.TrimSpace(r.Author) == "" {
        problems["author"] = "author is required"
    }
    return problems
}

// Comment handler
func handleComments(logger *logging.Logger, store *storage.CommentStore) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        userID := UserIDFromContext(ctx)

        switch r.Method {
        case http.MethodGet:
            comments, err := store.List(ctx)
            if err != nil {
                logger.Error(ctx, "failed to list comments",
                    "error", err,
                    "user_id", userID,
                )
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                return
            }

            // Map to response type
            resp := make([]commentResponse, len(comments))
            for i, c := range comments {
                resp[i] = commentResponse{
                    ID:        c.ID,
                    Content:   c.Content,
                    Author:    c.Author,
                    CreatedAt: c.CreatedAt,
                    UserID:    c.UserID,
                }
            }

            if err := encode(w, r, http.StatusOK, resp); err != nil {
                logger.Error(ctx, "failed to encode response",
                    "error", err,
                    "user_id", userID,
                )
                return
            }

        case http.MethodPost:
            req, problems, err := decodeValid[createCommentRequest](r)
            if err != nil {
                logger.Error(ctx, "failed to decode request",
                    "error", err,
                    "user_id", userID,
                )
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }
            if len(problems) > 0 {
                if err := encode(w, r, http.StatusBadRequest, problems); err != nil {
                    logger.Error(ctx, "failed to encode validation problems",
                        "error", err,
                        "user_id", userID,
                    )
                }
                return
            }

            comment, err := store.Create(ctx, storage.Comment{
                Content: req.Content,
                Author:  req.Author,
                UserID:  userID,
            })
            if err != nil {
                logger.Error(ctx, "failed to create comment",
                    "error", err,
                    "user_id", userID,
                )
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                return
            }

            resp := commentResponse{
                ID:        comment.ID,
                Content:   comment.Content,
                Author:    comment.Author,
                CreatedAt: comment.CreatedAt,
                UserID:    comment.UserID,
            }

            if err := encode(w, r, http.StatusCreated, resp); err != nil {
                logger.Error(ctx, "failed to encode response",
                    "error", err,
                    "user_id", userID,
                )
                return
            }

        default:
            http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        }
    })
}

// Add this to internal/api/handlers.go after the other handlers

// Single comment handler
func handleComment(logger *logging.Logger, store *storage.CommentStore) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        userID := UserIDFromContext(ctx)

        // Extract comment ID from URL
        commentID := strings.TrimPrefix(r.URL.Path, "/api/v1/comments/")
        if commentID == "" {
            http.Error(w, "Comment ID required", http.StatusBadRequest)
            return
        }

        switch r.Method {
        case http.MethodGet:
            comment, err := store.Get(ctx, commentID)
            if err != nil {
                if err == storage.ErrNotFound {
                    http.Error(w, "Comment not found", http.StatusNotFound)
                    return
                }
                logger.Error(ctx, "failed to get comment",
                    "error", err,
                    "comment_id", commentID,
                    "user_id", userID,
                )
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                return
            }

            resp := commentResponse{
                ID:        comment.ID,
                Content:   comment.Content,
                Author:    comment.Author,
                CreatedAt: comment.CreatedAt,
                UserID:    comment.UserID,
            }

            if err := encode(w, r, http.StatusOK, resp); err != nil {
                logger.Error(ctx, "failed to encode response",
                    "error", err,
                    "comment_id", commentID,
                    "user_id", userID,
                )
            }

        case http.MethodPut:
            req, problems, err := decodeValid[createCommentRequest](r)
            if err != nil {
                logger.Error(ctx, "failed to decode request",
                    "error", err,
                    "user_id", userID,
                )
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }
            if len(problems) > 0 {
                if err := encode(w, r, http.StatusBadRequest, problems); err != nil {
                    logger.Error(ctx, "failed to encode validation problems",
                        "error", err,
                        "user_id", userID,
                    )
                }
                return
            }

            // Verify the comment exists and belongs to the user
            existing, err := store.Get(ctx, commentID)
            if err != nil {
                if err == storage.ErrNotFound {
                    http.Error(w, "Comment not found", http.StatusNotFound)
                    return
                }
                logger.Error(ctx, "failed to get comment",
                    "error", err,
                    "comment_id", commentID,
                    "user_id", userID,
                )
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                return
            }

            if existing.UserID != userID {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }

            comment, err := store.Update(ctx, commentID, storage.Comment{
                Content: req.Content,
                Author:  req.Author,
                UserID:  userID,
            })
            if err != nil {
                logger.Error(ctx, "failed to update comment",
                    "error", err,
                    "comment_id", commentID,
                    "user_id", userID,
                )
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                return
            }

            resp := commentResponse{
                ID:        comment.ID,
                Content:   comment.Content,
                Author:    comment.Author,
                CreatedAt: comment.CreatedAt,
                UserID:    comment.UserID,
            }

            if err := encode(w, r, http.StatusOK, resp); err != nil {
                logger.Error(ctx, "failed to encode response",
                    "error", err,
                    "comment_id", commentID,
                    "user_id", userID,
                )
            }

        case http.MethodDelete:
            // Verify the comment exists and belongs to the user
            existing, err := store.Get(ctx, commentID)
            if err != nil {
                if err == storage.ErrNotFound {
                    http.Error(w, "Comment not found", http.StatusNotFound)
                    return
                }
                logger.Error(ctx, "failed to get comment",
                    "error", err,
                    "comment_id", commentID,
                    "user_id", userID,
                )
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                return
            }

            if existing.UserID != userID {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }

            if err := store.Delete(ctx, commentID); err != nil {
                logger.Error(ctx, "failed to delete comment",
                    "error", err,
                    "comment_id", commentID,
                    "user_id", userID,
                )
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                return
            }

            w.WriteHeader(http.StatusNoContent)

        default:
            http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        }
    })
}

// Login types
type loginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type loginResponse struct {
    Token     string `json:"token"`
    ExpiresIn int64  `json:"expires_in"`
}

func (r loginRequest) Valid(ctx context.Context) map[string]string {
    problems := make(map[string]string)
    if strings.TrimSpace(r.Username) == "" {
        problems["username"] = "username is required"
    }
    if strings.TrimSpace(r.Password) == "" {
        problems["password"] = "password is required"
    }
    return problems
}

// Login handler
func handleLogin(logger *logging.Logger, jwtManager *auth.JWTManager) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        req, problems, err := decodeValid[loginRequest](r)
        if err != nil {
            logger.Error(ctx, "failed to decode login request", "error", err)
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        if len(problems) > 0 {
            if err := encode(w, r, http.StatusBadRequest, problems); err != nil {
                logger.Error(ctx, "failed to encode validation problems", "error", err)
            }
            return
        }

        // In a real application, you would validate credentials against a database
        // This is just for demonstration
        if req.Username != "test" || req.Password != "test123" {
            logger.Warn(ctx, "invalid login attempt",
                "username", req.Username,
                "remote_addr", r.RemoteAddr,
            )
            http.Error(w, "Invalid credentials", http.StatusUnauthorized)
            return
        }

        token, err := jwtManager.GenerateToken(req.Username, "user")
        if err != nil {
            logger.Error(ctx, "failed to generate token", "error", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        resp := loginResponse{
            Token:     token,
            ExpiresIn: 24 * 60 * 60, // 24 hours in seconds
        }

        if err := encode(w, r, http.StatusOK, resp); err != nil {
            logger.Error(ctx, "failed to encode login response", "error", err)
            return
        }

        logger.Info(ctx, "successful login",
            "username", req.Username,
            "remote_addr", r.RemoteAddr,
        )
    })
}

// Health check handler
func handleHealthz(logger *logging.Logger) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if err := encode(w, r, http.StatusOK, map[string]string{
            "status": "ok",
            "time":   time.Now().UTC().Format(time.RFC3339),
        }); err != nil {
            logger.Error(r.Context(), "failed to encode health check response", "error", err)
        }
    })
}