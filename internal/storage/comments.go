// internal/storage/comments.go

package storage

import (
    "context"
    "errors"
    "sync"
    "time"
    "web-service/internal/util"
)

var (
    ErrNotFound = errors.New("comment not found")
)

type Comment struct {
    ID        string
    Content   string
    Author    string
    CreatedAt time.Time
    UserID    string    // Added to track who created the comment
}

type CommentStore struct {
    mu       sync.RWMutex
    comments map[string]Comment
}

func NewCommentStore() *CommentStore {
    return &CommentStore{
        comments: make(map[string]Comment),
    }
}

func (s *CommentStore) Create(ctx context.Context, c Comment) (Comment, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    select {
    case <-ctx.Done():
        return Comment{}, ctx.Err()
    default:
    }

    c.ID = util.GenerateID()
    c.CreatedAt = time.Now()
    s.comments[c.ID] = c
    return c, nil
}

func (s *CommentStore) List(ctx context.Context) ([]Comment, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    comments := make([]Comment, 0, len(s.comments))
    for _, c := range s.comments {
        comments = append(comments, c)
    }
    return comments, nil
}

func (s *CommentStore) Get(ctx context.Context, id string) (Comment, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    select {
    case <-ctx.Done():
        return Comment{}, ctx.Err()
    default:
    }

    comment, exists := s.comments[id]
    if !exists {
        return Comment{}, ErrNotFound
    }
    return comment, nil
}

func (s *CommentStore) Delete(ctx context.Context, id string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    if _, exists := s.comments[id]; !exists {
        return ErrNotFound
    }

    delete(s.comments, id)
    return nil
}

func (s *CommentStore) Update(ctx context.Context, id string, c Comment) (Comment, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    select {
    case <-ctx.Done():
        return Comment{}, ctx.Err()
    default:
    }

    existing, exists := s.comments[id]
    if !exists {
        return Comment{}, ErrNotFound
    }

    // Preserve creation metadata
    c.ID = existing.ID
    c.CreatedAt = existing.CreatedAt
    c.UserID = existing.UserID // Prevent user ID changes

    s.comments[id] = c
    return c, nil
}

// Optional: Add methods for querying comments

func (s *CommentStore) ListByUser(ctx context.Context, userID string) ([]Comment, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    var comments []Comment
    for _, c := range s.comments {
        if c.UserID == userID {
            comments = append(comments, c)
        }
    }
    return comments, nil
}

func (s *CommentStore) DeleteByUser(ctx context.Context, userID string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    for id, c := range s.comments {
        if c.UserID == userID {
            delete(s.comments, id)
        }
    }
    return nil
}

// Optional: Add a method to clean up old comments
func (s *CommentStore) DeleteOlderThan(ctx context.Context, age time.Duration) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    cutoff := time.Now().Add(-age)
    for id, c := range s.comments {
        if c.CreatedAt.Before(cutoff) {
            delete(s.comments, id)
        }
    }
    return nil
}

// Optional: Add a method to count comments
func (s *CommentStore) Count(ctx context.Context) (int, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    select {
    case <-ctx.Done():
        return 0, ctx.Err()
    default:
    }

    return len(s.comments), nil
}