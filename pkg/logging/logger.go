// pkg/logging/logger.go

package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

type Level int

const (
    DEBUG Level = iota
    INFO
    WARN
    ERROR
)

func (l Level) String() string {
    switch l {
    case DEBUG:
        return "DEBUG"
    case INFO:
        return "INFO"
    case WARN:
        return "WARN"
    case ERROR:
        return "ERROR"
    default:
        return "UNKNOWN"
    }
}

type Logger struct {
    out    io.Writer
    level  Level
}

type logEntry struct {
    Time       time.Time              `json:"time"`
    Level      string                 `json:"level"`
    Message    string                 `json:"message"`
    Caller     string                 `json:"caller,omitempty"`
    Fields     map[string]interface{} `json:"fields,omitempty"`
    StackTrace string                 `json:"stack_trace,omitempty"`
}

func NewLogger(out io.Writer) *Logger {
    if out == nil {
        out = os.Stdout
    }
    return &Logger{
        out:   out,
        level: INFO,
    }
}

func (l *Logger) SetLevel(level Level) {
    l.level = level
}

func (l *Logger) log(ctx context.Context, level Level, msg string, fields ...interface{}) {
    if level < l.level {
        return
    }

    entry := logEntry{
        Time:    time.Now(),
        Level:   level.String(),
        Message: msg,
        Fields:  make(map[string]interface{}),
    }

    // Add caller information
    if pc, file, line, ok := runtime.Caller(2); ok {
        if fn := runtime.FuncForPC(pc); fn != nil {
            entry.Caller = fmt.Sprintf("%s:%d", file, line)
        }
    }

    // Add context values if any
    if ctx != nil {
        if requestID, ok := ctx.Value("request_id").(string); ok {
            entry.Fields["request_id"] = requestID
        }
        if userID, ok := ctx.Value("user_id").(string); ok {
            entry.Fields["user_id"] = userID
        }
    }

    // Add additional fields
    for i := 0; i < len(fields)-1; i += 2 {
        if key, ok := fields[i].(string); ok {
            entry.Fields[key] = fields[i+1]
        }
    }

    // Add stack trace for errors
    if level == ERROR {
        buf := make([]byte, 1024)
        n := runtime.Stack(buf, false)
        entry.StackTrace = string(buf[:n])
    }

    // Encode and write the log entry
    if data, err := json.Marshal(entry); err == nil {
        l.out.Write(append(data, '\n'))
    }
}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...interface{}) {
    l.log(ctx, DEBUG, msg, fields...)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...interface{}) {
    l.log(ctx, INFO, msg, fields...)
}

func (l *Logger) Warn(ctx context.Context, msg string, fields ...interface{}) {
    l.log(ctx, WARN, msg, fields...)
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...interface{}) {
    l.log(ctx, ERROR, msg, fields...)
}

// Middleware to add request ID to context
func NewLoggingMiddleware(logger *Logger, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Generate request ID
        requestID := fmt.Sprintf("%d", time.Now().UnixNano())

        // Create new context with request ID
        ctx := context.WithValue(r.Context(), "request_id", requestID)

        // Create response writer wrapper to capture status code
        wrw := &responseWriter{
            ResponseWriter: w,
            status:        http.StatusOK,
        }

        // Log request
        logger.Info(ctx, "request started",
            "method", r.Method,
            "path", r.URL.Path,
            "request_id", requestID,
            "remote_addr", r.RemoteAddr,
        )

        startTime := time.Now()

        // Call next handler
        next.ServeHTTP(wrw, r.WithContext(ctx))

        // Log response
        logger.Info(ctx, "request completed",
            "method", r.Method,
            "path", r.URL.Path,
            "status", wrw.status,
            "duration_ms", time.Since(startTime).Milliseconds(),
            "request_id", requestID,
        )
    })
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
}

// Function to add trace ID to context
func NewGoogleTraceIDMiddleware(logger *Logger, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        traceID := r.Header.Get("X-Cloud-Trace-Context")
        if traceID == "" {
            traceID = fmt.Sprintf("trace-%d", time.Now().UnixNano())
        }

        ctx := context.WithValue(r.Context(), "trace_id", traceID)
        logger.Debug(ctx, "trace context added",
            "trace_id", traceID,
        )

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}