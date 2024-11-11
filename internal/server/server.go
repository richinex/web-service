// internal/server/server.go

package server

import (
    "context"
    "flag"
    "fmt"
    "io"
    "net"
    "net/http"
    "time"
    "web-service/internal/api"
    "web-service/internal/config"
    "web-service/internal/storage"
    "web-service/pkg/logging"
)

func Run(ctx context.Context, w io.Writer, args []string, getenv func(string) string) error {
    // Parse flags
    flags := flag.NewFlagSet(args[0], flag.ExitOnError)
    var (
        host = flags.String("host", "localhost", "Server host")
        port = flags.String("port", "8080", "Server port")
    )
    if err := flags.Parse(args[1:]); err != nil {
        return fmt.Errorf("parsing flags: %w", err)
    }

    // Initialize logger
    logger := logging.NewLogger(w)

    // Load config
    cfg, err := config.Load(getenv)
    if err != nil {
        return fmt.Errorf("loading config: %w", err)
    }

    // Initialize storage
    commentStore := storage.NewCommentStore()

    // Create server using api.NewServer
    handler := api.NewServer(
        logger,
        cfg,
        commentStore,
    )

    // Set up HTTP server
    httpServer := &http.Server{
        Addr:    net.JoinHostPort(*host, *port),
        Handler: handler,
    }

    // Channel to signal when the server is ready
    ready := make(chan struct{})

    // Create server listener manually so we can confirm it's ready
    listener, err := net.Listen("tcp", httpServer.Addr)
    if err != nil {
        return fmt.Errorf("failed to create listener: %w", err)
    }

    // Start server in a goroutine
    errChan := make(chan error, 1)
    go func() {
        logger.Info(ctx, "server starting", "addr", httpServer.Addr)

        // Signal that we're ready to accept connections
        close(ready)

        // Serve using the listener
        if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
            errChan <- fmt.Errorf("error serving: %w", err)
        }
        close(errChan)
    }()

    // Wait for server to be ready or for an error
    select {
    case <-ready:
        logger.Info(ctx, "server ready", "addr", httpServer.Addr)
    case err := <-errChan:
        return fmt.Errorf("server failed before becoming ready: %w", err)
    case <-time.After(5 * time.Second):
        return fmt.Errorf("timeout waiting for server to become ready")
    }

    // Wait for shutdown signal or error
    select {
    case err := <-errChan:
        return err
    case <-ctx.Done():
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        logger.Info(ctx, "shutting down server gracefully")
        if err := httpServer.Shutdown(shutdownCtx); err != nil {
            return fmt.Errorf("error shutting down server: %w", err)
        }
        return nil
    }
}